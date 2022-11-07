package canboat

import "errors"

var (
	ErrUnknownEnumType  = errors.New("unknown enum type given")
	ErrUnknownEnumValue = errors.New("unknown enum value given")
)

type LookupEnumerations []Enum

func (le LookupEnumerations) FindValue(enum string, value uint32) (EnumValue, error) {
	for _, e := range le {
		if e.Name != enum {
			continue
		}
		for _, v := range e.Values {
			if v.Value == value {
				return v, nil
			}
		}
		return EnumValue{}, ErrUnknownEnumValue
	}
	return EnumValue{}, ErrUnknownEnumType
}

func (le LookupEnumerations) Exists(enum string) bool {
	for _, e := range le {
		if e.Name == enum {
			return true
		}
	}
	return false
}

type Enum struct {
	Name   string      `json:"Name"`
	Values []EnumValue `json:"EnumValues"`
}

type EnumValue struct {
	Name  string `json:"Name"`
	Value uint32 `json:"Value"`
}

type LookupBitEnumerations []BitEnum

func (le LookupBitEnumerations) FindValue(enum string, value uint32) ([]BitEnumValue, error) {
	result := make([]BitEnumValue, 0)
	if value == 0 {
		return nil, nil
	}

	for _, e := range le {
		if e.Name != enum {
			continue
		}
		for _, v := range e.Values {
			if (value & (1 << v.Bit)) != 0 {
				result = append(result, v)
			}
		}
		if len(result) == 0 {
			return nil, ErrUnknownEnumValue
		}
	}
	if len(result) == 0 {
		return nil, ErrUnknownEnumType
	}
	return result, nil
}

func (le LookupBitEnumerations) Exists(enum string) bool {
	for _, e := range le {
		if e.Name == enum {
			return true
		}
	}
	return false
}

type BitEnum struct {
	Name   string         `json:"Name"`
	Values []BitEnumValue `json:"EnumBitValues"`
}

type BitEnumValue struct {
	Name string `json:"Name"`
	Bit  uint32 `json:"Bit"`
}

type LookupIndirectEnumerations []IndirectEnum

func (le LookupIndirectEnumerations) FindValue(enum string, value uint32, indirectValue uint32) (IndirectEnumValue, error) {
	for _, e := range le {
		if e.Name != enum {
			continue
		}
		for _, v := range e.Values {
			if v.Value == value && v.IndirectValue == indirectValue {
				return v, nil
			}
		}
		return IndirectEnumValue{}, ErrUnknownEnumValue
	}
	return IndirectEnumValue{}, ErrUnknownEnumType
}

func (le LookupIndirectEnumerations) Exists(enum string) bool {
	for _, e := range le {
		if e.Name == enum {
			return true
		}
	}
	return false
}

type IndirectEnum struct {
	Name   string              `json:"Name"`
	Values []IndirectEnumValue `json:"EnumValues"`
}

type IndirectEnumValue struct {
	Name          string `json:"Name"`
	IndirectValue uint32 `json:"Value1"`
	Value         uint32 `json:"Value2"`
}
