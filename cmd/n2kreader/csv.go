package main

import (
	"crypto/md5"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"github.com/aldas/go-nmea-client"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type csvPGNs []csvPGNFields

func writeCSV(field csvPGNFields, values []string) error {
	fileExists := false
	fi, err := os.Stat(field.fileName)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("csv file check failure, err: %s", err)
	}
	if fi != nil {
		fileExists = true
		if fi.IsDir() {
			return fmt.Errorf("csv file overlaps with directory, file: %s", field.fileName)
		}
	}

	var csvFile *os.File
	if fileExists {
		csvFile, err = os.OpenFile(field.fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	} else {
		csvFile, err = os.Create(field.fileName)
	}
	if err != nil {
		return err
	}
	defer csvFile.Close()

	csvwriter := csv.NewWriter(csvFile)

	fmt.Printf("fileExists: %v\n", fileExists)
	if !fileExists {
		if err := csvwriter.Write(append([]string{"time_ms"}, field.fields...)); err != nil {
			return fmt.Errorf("csv failed to write header, err: %s", err)
		}
	}
	if err := csvwriter.Write(values); err != nil {
		return fmt.Errorf("csv failed to write row, err: %s", err)
	}
	csvwriter.Flush()

	return nil
}

func (c csvPGNs) Match(pgn nmea.Message, now time.Time) ([]string, csvPGNFields, bool) {
	ok := false
	var found csvPGNFields
	for _, p := range c {
		if p.PGN == pgn.Header.PGN {
			found = p
			ok = true
			break
		}
	}
	if !ok {
		return nil, csvPGNFields{}, false
	}
	fields := make([]string, 0, len(found.fields)+1)

	for _, fID := range found.fields {
		v := ""
		switch fID {
		case "time_ms":
			v = strconv.FormatInt(now.UnixMilli(), 10)
		case "time_nano":
			v = strconv.FormatInt(now.UnixNano(), 10)
		default:
			fv, ok := pgn.Fields.FindByID(fID)
			if ok {
				switch vv := fv.Value.(type) {
				case string:
					v = vv
				case []byte:
					v = string(vv)
				default:
					ff, ok := fv.AsFloat64()
					if ok && !(math.IsInf(ff, 0) || math.IsNaN(ff)) {
						v = fmt.Sprintf("%.8g", ff)
					}
				}
			}
		}
		fields = append(fields, v)
	}
	if len(fields) <= 1 {
		return nil, csvPGNFields{}, false
	}
	return fields, found, true
}

type csvPGNFields struct {
	PGN      uint32
	fileName string
	fields   []string
}

func parseCSVFieldsRaw(raw string) ([]csvPGNFields, error) {
	// 129025:latitude,longitude;65280:manufacturerCode,industryCode
	result := make([]csvPGNFields, 0)
	raw = strings.TrimSpace(raw)
	parts := strings.Split(raw, ";")
	for _, p := range parts {
		pgnRaw, fieldsRaw, ok := strings.Cut(p, ":")
		if !ok {
			continue
		}
		pgn, err := strconv.ParseUint(strings.TrimSpace(pgnRaw), 10, 32)
		if err != nil {
			return nil, fmt.Errorf("csv fields: failed to parse PGN, err: %w", err)
		}

		tmpFields := make([]string, 0)
		for _, f := range strings.Split(fieldsRaw, ",") {
			f = strings.TrimSpace(f)
			if f == "" {
				continue
			}
			tmpFields = append(tmpFields, f)
		}
		if len(tmpFields) == 0 {
			continue
		}
		sort.Strings(tmpFields)
		hashBytes := md5.Sum([]byte(strings.Join(tmpFields, ",")))
		hash := hex.EncodeToString(hashBytes[:])

		tmp := csvPGNFields{
			PGN:      uint32(pgn),
			fileName: fmt.Sprintf("%v_%v.csv", pgn, hash),
			fields:   tmpFields,
		}
		result = append(result, tmp)
	}
	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}