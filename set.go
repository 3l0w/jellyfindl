package main

const present = ""

type Set struct {
	elements map[string]string
}

func NewSet() *Set {
	return &Set{make(map[string]string)}
}

func (s *Set) Add(element string) {
	s.elements[element] = present
}

func (s *Set) AddAll(elements []string) {
	for _, v := range elements {
		s.Add(v)
	}
}

func (s *Set) Remove(element string) {
	delete(s.elements, element)
}

func (s *Set) Contains(element string) bool {
	_, found := s.elements[element]
	return found
}

func (s *Set) Size() int {
	return len(s.elements)
}

func (s *Set) IsEmpty() bool {
	return s.Size() == 0
}

func (s *Set) Clear() {
	s.elements = make(map[string]string)
}

func (s *Set) Values() []string {
	values := make([]string, 0, s.Size())
	for value := range s.elements {
		values = append(values, value)
	}
	return values
}

func (s *Set) Toggle(val string) bool {
	if s.Contains(val) {
		s.Remove(val)
		return false
	} else {
		s.Add(val)
		return true
	}
}
