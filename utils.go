package ECMSLogger

import (
	"errors"
	"os"
	"path"
	"reflect"
	"strconv"
	"strings"
)

func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// StructFields reflects on a struct and returns the values of fields with `tagName` tags,
// or a map[string]interface{} and returns the keys.
func StructFields(values interface{}, tagName string, except []string) ([]string, error) {
	v := reflect.ValueOf(values)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	fields := []string{}
	if v.Kind() == reflect.Struct {
		for i := 0; i < v.NumField(); i++ {
			field := v.Type().Field(i).Tag.Get(tagName)
			if field != "" && !StringInSlice(field, except) {
				fields = append(fields, field)
			}
		}
		return fields, nil
	}
	if v.Kind() == reflect.Map {
		for _, keyv := range v.MapKeys() {
			if !StringInSlice(keyv.String(), except) {
				fields = append(fields, keyv.String())
			}
		}
		return fields, nil
	}
	return []string{}, errors.New("DBFields requires a struct or a map, found: " + v.Kind().String())
}

func ParseSize(str string) int64 {
	low := strings.ToLower(str)
	last := low[len(low)-1]
	mult := int64(1)
	switch last {
	case 'b':
		mult = int64(1)
	case 'k':
		mult = int64(1024)
	case 'm':
		mult = int64(1024 * 1024)
	case 'g':
		mult = int64(1024 * 1024 * 1024)
	default:
		return 0
	}
	a, err := strconv.ParseInt(low[0:len(low)-1], 10, 64)
	if err != nil {
		return 0
	}
	return a * mult
}

func CheckTouch(dir string) error {
	filename := path.Join(dir, "dummy")
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	x := []byte("empty")
	n, err := file.Write(x)
	if err != nil {
		return err
	}
	if n != len(x) {
		return errors.New("Writeen bytes are not equal to data length")
	}
	err = file.Sync()
	if err != nil {
		return err
	}
	err = os.Remove(filename)
	return err
}
