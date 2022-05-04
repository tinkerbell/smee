package unit

type Builder struct {
	sections []*SectionBuilder
}

func (b *Builder) AddSection(name string, lines ...string) *SectionBuilder {
	s := new(SectionBuilder).Reset(name).AddLines(lines...)
	b.sections = append(b.sections, s)

	return s
}

func (b *Builder) Len() (sum int) {
	for i, s := range b.sections {
		if i > 0 {
			sum += 1
		}
		sum += s.Len()
	}

	return
}

func (b *Builder) MarshalText() ([]byte, error) {
	buf := make([]byte, 0, b.Len())
	for i, s := range b.sections {
		if i > 0 {
			buf = append(buf, '\n') // blank line between sections for readability
		}
		buf = append(buf, s.buf...)
	}

	return buf, nil
}

func (b *Builder) Bytes() []byte {
	buf, _ := b.MarshalText()

	return buf
}

type SectionBuilder struct {
	buf []byte
}

func (s *SectionBuilder) Add(key, value string) *SectionBuilder {
	// TODO: Validate key.
	// TODO: Handle multiline values.
	s.buf = append(append(append(append(s.buf, key...), '='), value...), '\n')

	return s // for chaining
}

func (s *SectionBuilder) AddLines(lines ...string) *SectionBuilder {
	for _, line := range lines {
		s.buf = append(append(s.buf, line...), '\n')
	}

	return s
}

func (s *SectionBuilder) AddComment(comment string) *SectionBuilder {
	// TODO: Handle multiline comments.
	s.buf = append(append(append(s.buf, "# "...), comment...), '\n')

	return s // for chaining
}

func (s *SectionBuilder) Len() int {
	return len(s.buf)
}

func (s *SectionBuilder) Reset(name string) *SectionBuilder {
	// TODO: Validate name.
	s.buf = append(append(append(s.buf[:0], '['), name...), "]\n"...)

	return s // for chaining
}
