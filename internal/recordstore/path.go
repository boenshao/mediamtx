package recordstore

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bluenviron/mediamtx/internal/conf"
)

func leadingZeros(v int, size int) string {
	out := strconv.FormatInt(int64(v), 10)
	if len(out) >= size {
		return out
	}

	out2 := ""
	for i := 0; i < (size - len(out)); i++ {
		out2 += "0"
	}

	return out2 + out
}

// PathAddExtension adds the file extension to the path.
func PathAddExtension(path string, format conf.RecordFormat) string {
	switch format {
	case conf.RecordFormatMPEGTS:
		return path + ".ts"

	default:
		return path + ".mp4"
	}
}

// CommonPath returns the common path between all segments with given recording path.
func CommonPath(v string) string {
	common := ""
	remaining := v

	for {
		i := strings.IndexAny(remaining, "\\/")
		if i < 0 {
			break
		}

		var part string
		part, remaining = remaining[:i+1], remaining[i+1:]

		if strings.Contains(part, "%") {
			break
		}

		common += part
	}

	if len(common) > 0 {
		common = common[:len(common)-1]
	}

	return common
}

// Path is a path of a recording segment.
type Path struct {
	Start time.Time
	Path  string
}

// Decode decodes a Path.
func (p *Path) Decode(format string, v string) bool {
	re := format

	for _, ch := range []uint8{
		'\\',
		'.',
		'+',
		'*',
		'?',
		'^',
		'$',
		'(',
		')',
		'[',
		']',
		'{',
		'}',
		'|',
	} {
		re = strings.ReplaceAll(re, string(ch), "\\"+string(ch))
	}

	re = strings.ReplaceAll(re, "%path", "(.*?)")
	re = strings.ReplaceAll(re, "%Y", "([0-9]{4})")
	re = strings.ReplaceAll(re, "%m", "([0-9]{2})")
	re = strings.ReplaceAll(re, "%d", "([0-9]{2})")
	re = strings.ReplaceAll(re, "%H", "([0-9]{2})")
	re = strings.ReplaceAll(re, "%M", "([0-9]{2})")
	re = strings.ReplaceAll(re, "%S", "([0-9]{2})")
	re = strings.ReplaceAll(re, "%f", "([0-9]{6})")
	re = strings.ReplaceAll(re, "%s", "([0-9]{10})")
	re = strings.ReplaceAll(re, "%z", "([+-][0-9]{4})")
	r := regexp.MustCompile(re)

	var groupMapping []string
	cur := format
	for {
		i := strings.Index(cur, "%")
		if i < 0 {
			break
		}

		cur = cur[i:]

		for _, va := range []string{
			"%path",
			"%Y",
			"%m",
			"%d",
			"%H",
			"%M",
			"%S",
			"%f",
			"%s",
			"%z",
		} {
			if strings.HasPrefix(cur, va) {
				groupMapping = append(groupMapping, va)
			}
		}

		cur = cur[1:]
	}

	matches := r.FindStringSubmatch(v)
	if matches == nil {
		return false
	}

	values := make(map[string]string)

	for i, match := range matches[1:] {
		values[groupMapping[i]] = match
	}

	var year int
	var month time.Month = 1
	day := 1
	var hour int
	var minute int
	var second int
	var micros int
	var unixSec int64 = -1
	var loc *time.Location = time.Local

	for k, v := range values {
		switch k {
		case "%path":
			p.Path = v

		case "%Y":
			tmp, _ := strconv.ParseInt(v, 10, 64)
			year = int(tmp)

		case "%m":
			tmp, _ := strconv.ParseInt(v, 10, 64)
			month = time.Month(int(tmp))

		case "%d":
			tmp, _ := strconv.ParseInt(v, 10, 64)
			day = int(tmp)

		case "%H":
			tmp, _ := strconv.ParseInt(v, 10, 64)
			hour = int(tmp)

		case "%M":
			tmp, _ := strconv.ParseInt(v, 10, 64)
			minute = int(tmp)

		case "%S":
			tmp, _ := strconv.ParseInt(v, 10, 64)
			second = int(tmp)

		case "%f":
			tmp, _ := strconv.ParseInt(v, 10, 64)
			micros = int(tmp)

		case "%s":
			unixSec, _ = strconv.ParseInt(v, 10, 64)

		case "%z":
			loc = time.FixedZone("fromFilename", strToOffset(v))
		}
	}

	if unixSec > 0 {
		p.Start = time.Unix(unixSec, 0)
	} else {
		p.Start = time.Date(year, month, day, hour, minute, second, micros*1000, loc)
	}

	return true
}

// Encode encodes a path.
func (p Path) Encode(format string) string {
	format = strings.ReplaceAll(format, "%path", p.Path)
	format = strings.ReplaceAll(format, "%Y", strconv.FormatInt(int64(p.Start.Year()), 10))
	format = strings.ReplaceAll(format, "%m", leadingZeros(int(p.Start.Month()), 2))
	format = strings.ReplaceAll(format, "%d", leadingZeros(p.Start.Day(), 2))
	format = strings.ReplaceAll(format, "%H", leadingZeros(p.Start.Hour(), 2))
	format = strings.ReplaceAll(format, "%M", leadingZeros(p.Start.Minute(), 2))
	format = strings.ReplaceAll(format, "%S", leadingZeros(p.Start.Second(), 2))
	format = strings.ReplaceAll(format, "%f", leadingZeros(p.Start.Nanosecond()/1000, 6))
	format = strings.ReplaceAll(format, "%s", strconv.FormatInt(p.Start.Unix(), 10))
	_, offset := p.Start.Zone()
	format = strings.ReplaceAll(format, "%z", offsetToStr(offset))
	return format
}

func offsetToStr(offset int) string {
	z := "+"
	if offset < 0 {
		offset *= -1
		z = "-"
	}

	h := offset / 3600
	m := (offset % 3600) / 60
	z += leadingZeros(h, 2) + leadingZeros(m, 2)

	return z
}

func strToOffset(z string) int {
	h, _ := strconv.ParseInt(z[1:3], 10, 64)
	m, _ := strconv.ParseInt(z[3:5], 10, 64)

	offset := int(h)*3600 + int(m)*60
	if z[0] == '-' {
		offset *= -1
	}

	return offset
}
