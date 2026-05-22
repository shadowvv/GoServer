package wordFilter

import (
	"bufio"
	"os"
	"sync/atomic"

	"github.com/importcjj/sensitive"
)

func InitWordFilterService(path string) {
	service, err := newWordFilter(path)
	if err != nil {
		panic(err)
	}
	wordFilterService = service
}

func HasSensitive(text string) bool {
	return wordFilterService.HasSensitive(text)
}

func Reload() error {
	return wordFilterService.Reload()
}

func Find(text string) []string {
	return wordFilterService.Find(text)
}

func Replace(text string) string {
	return wordFilterService.Replace(text)
}

var wordFilterService *WordFilter

type WordFilter struct {
	matcher atomic.Value // 并发安全
	path    string
}

// 初始化
func newWordFilter(path string) (*WordFilter, error) {
	wf := &WordFilter{path: path}
	if err := wf.load(); err != nil {
		return nil, err
	}
	return wf, nil
}

// 读取词库文件 + 构建 matcher
func (wf *WordFilter) load() error {
	f := sensitive.New()

	file, err := os.Open(wf.path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		f.AddWord(line)
	}
	if err = scanner.Err(); err != nil {
		return err
	}

	wf.matcher.Store(f)
	return nil
}

// ========== 对外查询功能 ===========

func (wf *WordFilter) HasSensitive(text string) bool {
	m := wf.matcher.Load().(*sensitive.Filter)
	find, _ := m.FindIn(text)
	return find
}

func (wf *WordFilter) Find(text string) []string {
	m := wf.matcher.Load().(*sensitive.Filter)
	return m.FindAll(text)
}

func (wf *WordFilter) Replace(text string) string {
	m := wf.matcher.Load().(*sensitive.Filter)
	return m.Replace(text, '*')
}

// GM 手动 reload
func (wf *WordFilter) Reload() error {
	return wf.load()
}

// 内部：读取所有行
func (wf *WordFilter) readAll() ([]string, error) {
	file, err := os.Open(wf.path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var list []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			list = append(list, line)
		}
	}
	return list, scanner.Err()
}
