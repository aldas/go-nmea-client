package main

import (
	"crypto/md5"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"github.com/aldas/go-nmea-client"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
)

type csvPGNs []csvPGNFields

func writeCSV(cpf csvPGNFields, values []string) error {
	fileExists := false
	fi, err := os.Stat(cpf.fileName)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("csv file check failure, err: %s", err)
	}
	if fi != nil {
		fileExists = true
		if fi.IsDir() {
			return fmt.Errorf("csv file overlaps with directory, file: %s", cpf.fileName)
		}
	}

	var csvFile *os.File
	if fileExists {
		csvFile, err = os.OpenFile(cpf.fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	} else {
		csvFile, err = os.Create(cpf.fileName)
	}
	if err != nil {
		return err
	}
	defer csvFile.Close()

	csvwriter := csv.NewWriter(csvFile)

	if !fileExists {
		if err := csvwriter.Write(cpf.names); err != nil {
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
		switch fID.name {
		case "_time":
			tmpNow := now
			if fID.truncate > 0 {
				tmpNow = now.Truncate(fID.truncate)
			}
			v = strconv.FormatInt(tmpNow.Unix(), 10)
		case "_time_ms":
			tmpNow := now
			if fID.truncate > 0 {
				tmpNow = now.Truncate(fID.truncate)
			}
			v = strconv.FormatInt(tmpNow.UnixMilli(), 10)
		case "_time_nano":
			tmpNow := now
			if fID.truncate > 0 {
				tmpNow = now.Truncate(fID.truncate)
			}
			v = strconv.FormatInt(tmpNow.UnixNano(), 10)
		case "_src":
			v = strconv.FormatInt(int64(pgn.Header.Source), 10)
		case "_dst":
			v = strconv.FormatInt(int64(pgn.Header.Destination), 10)
		case "_prio":
			v = strconv.FormatInt(int64(pgn.Header.Priority), 10)
		default:
			fv, ok := pgn.Fields.FindByID(fID.name)
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
	names    []string
	fields   []field
}

type field struct {
	name     string
	truncate time.Duration
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

		tmpNames := make([]string, 0)
		tmpFields := make([]field, 0)
		for _, f := range strings.Split(fieldsRaw, ",") {
			f = strings.TrimSpace(f)
			if f == "" {
				continue
			}
			var trunc time.Duration
			if strings.HasPrefix(f, "_time") {
				start := strings.IndexByte(f, '(')
				end := strings.LastIndexByte(f, ')')
				if start != -1 && start+1 < end {
					if tRaw, err := time.ParseDuration(f[start+1 : end]); err != nil {
						return nil, fmt.Errorf("csv fields: invalid _time format, err: %w", err)
					} else {
						trunc = tRaw
					}
				}
				f = f[0:start]
			}
			tmpFields = append(tmpFields, field{
				name:     f,
				truncate: trunc,
			})
			tmpNames = append(tmpNames, f)
		}
		if len(tmpNames) == 0 {
			continue
		}

		hashBytes := md5.Sum([]byte(strings.Join(tmpNames, ",")))
		hash := hex.EncodeToString(hashBytes[:])

		tmp := csvPGNFields{
			PGN:      uint32(pgn),
			fileName: fmt.Sprintf("%v_%v.csv", pgn, hash),
			names:    tmpNames,
			fields:   tmpFields,
		}
		result = append(result, tmp)
	}
	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}
