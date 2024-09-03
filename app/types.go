package main

type LabelArray []string

type Request struct {
	header          HeaderOptions
	questionSection []QuestionSection
}

type QSandASPair struct {
	qs []byte
	as []byte
}

//type SafeMap struct {
//	mu sync.Mutex
//	m  map[int]QSandASPair
//}
//
//func (s *SafeMap) Set(key int, value QSandASPair) {
//	s.mu.Lock()
//	defer s.mu.Unlock()
//	s.m[key] = value
//}
//
//func (s *SafeMap) Get(key int) (QSandASPair, bool) {
//	s.mu.Lock()
//	defer s.mu.Unlock()
//	value, exists := s.m[key]
//	return value, exists
//}
