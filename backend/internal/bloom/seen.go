package bloom

import "hash/fnv"

type Filter struct {
	bits []uint64
	k    int
}

func New(bits int) *Filter {
	if bits < 64 {
		bits = 64
	}
	n := (bits + 63) / 64
	return &Filter{bits: make([]uint64, n), k: 3}
}

func (f *Filter) Add(key string) {
	for i := 0; i < f.k; i++ {
		idx, shift := f.pos(key, i)
		f.bits[idx] |= 1 << shift
	}
}

func (f *Filter) MayContain(key string) bool {
	for i := 0; i < f.k; i++ {
		idx, shift := f.pos(key, i)
		if f.bits[idx]&(1<<shift) == 0 {
			return false
		}
	}
	return true
}

func (f *Filter) pos(key string, i int) (int, uint) {
	h := fnv.New64a()
	_, _ = h.Write([]byte{byte(i)})
	_, _ = h.Write([]byte(key))
	v := h.Sum64()
	bits := uint64(len(f.bits)) * 64
	slot := v % bits
	return int(slot / 64), uint(slot % 64)
}
