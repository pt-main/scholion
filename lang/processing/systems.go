package processing

import (
	"fmt"
	"strings"
)

// -> ok (is 'has bufer')
func (s *System) AppendToBuffer(str, name string) bool {
	buf, ok := s.Buffers[name]
	if !ok {
		return ok
	}
	buf = append(buf, str)
	s.Buffers[name] = buf
	return true
}

// -> ok (is 'has bufer')
func (s *System) PrependToBuffer(str, name string) bool {
	buf, ok := s.Buffers[name]
	if !ok {
		return ok
	}
	buf = append([]string{str}, buf...)
	s.Buffers[name] = buf
	return true
}

func (s *System) HasBuffer(name string) bool {
	_, has := s.Buffers[name]
	return has
}

// create buffer.
//
// -> name
func (s *System) NewBuffer() string {
	name := fmt.Sprintf("__tmp_buf_n%v", s.bufCount)
	s.bufCount += 1
	s.Buffers[name] = []string{}
	return name
}

// -> ok (is 'has bufer')
func (s *System) ClearBuffer(name string) bool {
	if !s.HasBuffer(name) {
		return false
	}
	s.Buffers[name] = []string{}
	return true
}

// -> ok (is 'has bufer')
func (s *System) DeleteBuffer(name string) bool {
	if !s.HasBuffer(name) {
		return false
	}
	delete(s.Buffers, name)
	return true
}

// -> ok (is 'has bufer')
func (s *System) GetBuffer(name string) (string, bool) {
	buf, ok := s.Buffers[name]
	if !ok {
		return "", false
	}
	res := strings.Join(buf, "")
	return res, true
}

// -> ok (is 'has bufer')
func (s *System) PasteBuffer(name string, p *Processor) bool {
	res, ok := s.GetBuffer(name)
	if !ok {
		return false
	}
	p.Code = append(p.Code, res)
	return true
}

func (s *System) SkipRawTokens(n uint) {
	s.skipRawTokens += n
}
