package st

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

func CompleteFileName(str string, percent string, sharp []string) []string {
	envval := regexp.MustCompile("[$]([a-zA-Z]+)")
	if envval.MatchString(str) {
		efs := envval.FindStringSubmatch(str)
		if len(efs) >= 2 {
			val := os.Getenv(strings.ToUpper(efs[1]))
			if val != "" {
				str = strings.Replace(str, efs[0], val, 1)
			}
		}
	}
	if strings.HasPrefix(str, "~") {
		home := os.Getenv("HOME")
		if home != "" {
			str = strings.Replace(str, "~", home, 1)
		}
	}
	if strings.Contains(str, "%") && percent != "" {
		str = strings.Replace(str, "%:h", filepath.Dir(percent), 1)
		str = strings.Replace(str, "%<", PruneExt(percent), 1)
		str = strings.Replace(str, "%", percent, 1)
	}
	if len(sharp) > 0 {
		sh := regexp.MustCompile("#([0-9]+)")
		if sh.MatchString(str) {
			sfs := sh.FindStringSubmatch(str)
			if len(sfs) >= 2 {
				tmp, err := strconv.ParseInt(sfs[1], 10, 64)
				if err == nil && int(tmp) < len(sharp) {
					str = strings.Replace(str, sfs[0], sharp[int(tmp)], 1)
				}
			}
		}
	}
	lis := strings.Split(str, " ")
	path := lis[len(lis)-1]
	if !filepath.IsAbs(path) {
		path = filepath.Join(filepath.Dir(percent), path)
	}
	var err error
	var completes []string
	tmp, err := filepath.Glob(path + "*")
	if err != nil || len(tmp) == 0 {
		completes = make([]string, 0)
	} else {
		completes = make([]string, len(tmp))
		for i := 0; i < len(tmp); i++ {
			stat, err := os.Stat(tmp[i])
			if err != nil {
				continue
			}
			if stat.IsDir() {
				tmp[i] += string(os.PathSeparator)
			}
			lis[len(lis)-1] = tmp[i]
			completes[i] = strings.Join(lis, " ")
		}
	}
	return completes
}
