package util

import (
	"time"
	"bytes"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sort"
)

const (
	ATOM = "2006-01-02T15:04:05+08:00"
)


func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}


func GetAtomTime() string {
	t := time.Now().Unix()
	return time.Unix(t, 0).Format(ATOM)
}



// StringValue returns the string representation of a value.
func StringValue(i interface{}) string {
	var buf bytes.Buffer
	stringValue(reflect.ValueOf(i), &buf)
	return buf.String()
}

// stringValue will recursively walk value v to build a textual
// representation of the value.
func stringValue(v reflect.Value, buf *bytes.Buffer) {
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Struct:
		strtype := v.Type().String()
		if strtype == "time.Time" {
			fmt.Fprintf(buf, "%s", v.Interface())
			break
		} else if strings.HasPrefix(strtype, "io.") {
			buf.WriteString("<buffer>")
			break
		}
		names := []string{}
		for i := 0; i < v.Type().NumField(); i++ {
			name := v.Type().Field(i).Name
			f := v.Field(i)
			if name[0:1] == strings.ToLower(name[0:1]) {
				continue // ignore unexported fields
			}
			if (f.Kind() == reflect.Ptr || f.Kind() == reflect.Slice || f.Kind() == reflect.Map) && f.IsNil() {
				continue // ignore unset fields
			}
			names = append(names, name)
		}
		sort.Strings(names)
		for _, n := range names {
			val := v.FieldByName(n)
			buf.WriteString(strings.ToLower(n))
			stringValue(val,buf)
		}
	case reflect.Slice:
		nl, id2 := "", ""
		for i := 0; i < v.Len(); i++ {
			buf.WriteString(id2)
			stringValue(v.Index(i), buf)
			if i < v.Len()-1 {
				buf.WriteString("," + nl)
			}
		}
	case reflect.Map:
		for _, k := range v.MapKeys() {
			stringValue(v.MapIndex(k) ,buf)
		}
	default:
		format := "%v"
		switch v.Interface().(type) {
			case string:
			format = "%s"
			case io.ReadSeeker, io.Reader:
			format = "buffer(%p)"
		}
		fmt.Fprintf(buf, format, v.Interface())
	}
}