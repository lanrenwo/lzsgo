package lzsgo

import (
	"errors"
	"unsafe"
)

const (
	htSize = 65536
	EINVAL = 22
	EFBIG  = 27

	ErrEFBIG   = "content too large"
	ErrEINVAL  = "invalid argument"
	ErrZero    = "result length is zero"
	ErrUnknown = "unknown error"
)

var HashTable [htSize]uint16

func init() {
	for i := 0; i < htSize; i++ {
		HashTable[i] = 65535
	}
}

func Compress(src []byte, dst []byte) (int, error) {
	n := int(lzsCompress(&dst[0], int32(cap(dst)), &src[0], int32(len(src))))
	return n, parseErr(n)
}

func Uncompress(src []byte, dst []byte) (int, error) {
	n := int(lzsDecompress(&dst[0], int32(cap(dst)), &src[0], int32(len(src))))
	return n, parseErr(n)
}

func parseErr(n int) error {
	switch {
	case n > 0:
		return nil
	case n == -EFBIG:
		return errors.New(ErrEFBIG)
	case n == -EINVAL:
		return errors.New(ErrEINVAL)
	case n == 0:
		return errors.New(ErrZero)
	default:
		return errors.New(ErrUnknown)
	}
}

func lzsCompress(dst *uint8, dstlen int32, src *uint8, srclen int32) int32 {
	var length int32
	var offset int32
	var inpos int32 = 0
	var outpos int32 = 0
	var longest_match_len uint16
	var hofs uint16
	var longest_match_ofs uint16
	var hash uint16
	var outbits uint32 = 0
	var nr_outbits int32 = 0
	var hash_table [htSize]uint16 = HashTable
	var hash_chain [2048]uint16
	var vv int32
	if srclen > htSize {
		return -EFBIG
	}

	for inpos < srclen-2 {
		hash = *(*uint16)(unsafe.Pointer((*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(src)) + uintptr(inpos)))))
		hofs = hash_table[hash]
		hash_chain[inpos&2047] = hofs
		hash_table[hash] = uint16(inpos)
		if int32(hofs) == 65535 || int32(hofs)+2048 <= inpos {
			outbits <<= 9
			outbits |= uint32(*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(src)) + uintptr(inpos))))
			nr_outbits += 9
			{
				nr_outbits -= 8
				if outpos == dstlen {
					return -EFBIG
				}
				vv = outpos
				outpos++
				*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(dst)) + uintptr(vv))) = uint8(outbits >> nr_outbits)
			}
			if nr_outbits >= 8 {
				nr_outbits -= 8
				if outpos == dstlen {
					return -EFBIG
				}
				vv = outpos
				outpos++
				*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(dst)) + uintptr(vv))) = uint8(outbits >> nr_outbits)
			}
			inpos++
			continue
		}

		longest_match_len = 2
		longest_match_ofs = hofs
		for ; int32(hofs) != 65535 && int32(hofs)+2048 > inpos; hofs = hash_chain[hofs&2047] {
			if !(memcmp(uintptr(unsafe.Pointer(src))+uintptr(hofs+2), uintptr(unsafe.Pointer(src))+uintptr(inpos+2), uint64(int32(longest_match_len-1))) != 0) {
				longest_match_ofs = hofs
				for {
					longest_match_len++
					if int32(longest_match_len)+inpos == srclen {
						goto got_match
					}
					if !(int32(*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(src)) + uintptr(int32(longest_match_len)+inpos)))) == int32(*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(src)) + uintptr(int32(longest_match_len)+int32(hofs)))))) {
						break
					}
				}
			}
		}
	got_match:
		offset = inpos - int32(longest_match_ofs)
		length = int32(longest_match_len)

		if offset < 128 {
			outbits <<= 9
			outbits |= uint32(384 | offset)
			nr_outbits += 9

			nr_outbits -= 8
			if outpos == dstlen {
				return -EFBIG
			}
			vv = outpos
			outpos++
			*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(dst)) + uintptr(vv))) = uint8(outbits >> nr_outbits)

			if nr_outbits >= 8 {
				nr_outbits -= 8
				if outpos == dstlen {
					return -EFBIG
				}
				vv = outpos
				outpos++
				*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(dst)) + uintptr(vv))) = uint8(outbits >> nr_outbits)
			}
		} else {
			outbits <<= 13
			outbits |= uint32(4096 | offset)
			nr_outbits += 13
			nr_outbits -= 8
			if outpos == dstlen {
				return -EFBIG
			}
			vv = outpos
			outpos++
			*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(dst)) + uintptr(vv))) = uint8(outbits >> nr_outbits)
			if nr_outbits >= 8 {
				nr_outbits -= 8
				if outpos == dstlen {
					return -EFBIG
				}
				vv = outpos
				outpos++
				*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(dst)) + uintptr(vv))) = uint8(outbits >> nr_outbits)
			}
		}
		if length < 5 {
			outbits <<= 2
			outbits |= uint32(length - 2)
			nr_outbits += 2
			if false {
				nr_outbits -= 8
				if outpos == dstlen {
					return -EFBIG
				}
				vv = outpos
				outpos++
				*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(dst)) + uintptr(vv))) = uint8(outbits >> nr_outbits)
			}
			if nr_outbits >= 8 {
				nr_outbits -= 8
				if outpos == dstlen {
					return -EFBIG
				}
				vv = outpos
				outpos++
				*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(dst)) + uintptr(vv))) = uint8(outbits >> nr_outbits)
			}
		} else if length < 8 {
			outbits <<= 4
			outbits |= uint32(length + 7)
			nr_outbits += 4
			if false {
				nr_outbits -= 8
				if outpos == dstlen {
					return -EFBIG
				}
				vv = outpos
				outpos++
				*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(dst)) + uintptr(vv))) = uint8(outbits >> nr_outbits)
			}
			if nr_outbits >= 8 {
				nr_outbits -= 8
				if outpos == dstlen {
					return -EFBIG
				}
				vv = outpos
				outpos++
				*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(dst)) + uintptr(vv))) = uint8(outbits >> nr_outbits)
			}
		} else {
			length += 7
			for length >= 30 {
				outbits <<= 8
				outbits |= 255
				nr_outbits += 8
				if false {
					nr_outbits -= 8
					if outpos == dstlen {
						return -EFBIG
					}
					vv = outpos
					outpos++
					*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(dst)) + uintptr(vv))) = uint8(outbits >> nr_outbits)
				}
				if nr_outbits >= 8 {
					nr_outbits -= 8
					if outpos == dstlen {
						return -EFBIG
					}
					vv = outpos
					outpos++
					*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(dst)) + uintptr(vv))) = uint8(outbits >> nr_outbits)
				}
				length -= 30
			}
			if length >= 15 {
				outbits <<= 8
				outbits |= uint32(int32(240) + length - 15)
				nr_outbits += 8
				if false {
					nr_outbits -= 8
					if outpos == dstlen {
						return -EFBIG
					}
					vv = outpos
					outpos++
					*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(dst)) + uintptr(vv))) = uint8(outbits >> nr_outbits)
				}
				if nr_outbits >= 8 {
					nr_outbits -= 8
					if outpos == dstlen {
						return -EFBIG
					}
					vv = outpos
					outpos++
					*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(dst)) + uintptr(vv))) = uint8(outbits >> nr_outbits)
				}
			} else {
				outbits <<= 4
				outbits |= uint32(length)
				nr_outbits += 4
				if false {
					nr_outbits -= 8
					if outpos == dstlen {
						return -EFBIG
					}
					vv = outpos
					outpos++
					*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(dst)) + uintptr(vv))) = uint8(outbits >> nr_outbits)
				}
				if nr_outbits >= 8 {
					nr_outbits -= 8
					if outpos == dstlen {
						return -EFBIG
					}
					vv = outpos
					outpos++
					*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(dst)) + uintptr(vv))) = uint8(outbits >> nr_outbits)
				}
			}
		}
		if inpos+int32(longest_match_len) >= srclen-2 {
			inpos += int32(longest_match_len)
			break
		}
		inpos++
		for func() (_cgo_ret uint16) {
			_cgo_addr := &longest_match_len
			*_cgo_addr--
			return *_cgo_addr
		}() != 0 {
			hash = *(*uint16)(unsafe.Pointer((*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(src)) + uintptr(inpos)))))
			hash_chain[inpos&2047] = hash_table[hash]
			hash_table[hash] = uint16(func() (_cgo_ret int32) {
				_cgo_addr := &inpos
				_cgo_ret = *_cgo_addr
				*_cgo_addr++
				return
			}())
		}
	}

	if inpos == srclen-2 {
		hash = *(*uint16)(unsafe.Pointer((*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(src)) + uintptr(inpos)))))
		// hofs = c.get(hash)
		// hofs = hash_table[hash]
		hofs = hash_table[hash]
		if int32(hofs) != 65535 && int32(hofs)+2048 > inpos {
			offset = inpos - int32(hofs)
			if offset < 128 {
				{
					outbits <<= 9
					outbits |= uint32(384 | offset)
					nr_outbits += 9
					nr_outbits -= 8
					if outpos == dstlen {
						return -EFBIG
					}
					vv = outpos
					outpos++
					*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(dst)) + uintptr(vv))) = uint8(outbits >> nr_outbits)
					if nr_outbits >= 8 {
						nr_outbits -= 8
						if outpos == dstlen {
							return -EFBIG
						}
						vv = outpos
						outpos++
						*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(dst)) + uintptr(vv))) = uint8(outbits >> nr_outbits)
					}
				}
			} else {
				{
					outbits <<= 13
					outbits |= uint32(4096 | offset)
					nr_outbits += 13
					nr_outbits -= 8
					if outpos == dstlen {
						return -EFBIG
					}
					vv = outpos
					outpos++
					*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(dst)) + uintptr(vv))) = uint8(outbits >> nr_outbits)
					if nr_outbits >= 8 {
						nr_outbits -= 8
						if outpos == dstlen {
							return -EFBIG
						}
						vv = outpos
						outpos++
						*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(dst)) + uintptr(vv))) = uint8(outbits >> nr_outbits)
					}
				}
			}
			{
				outbits <<= 2
				outbits |= 0
				nr_outbits += 2
				if false {
					nr_outbits -= 8
					if outpos == dstlen {
						return -EFBIG
					}
					vv = outpos
					outpos++
					*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(dst)) + uintptr(vv))) = uint8(outbits >> nr_outbits)
				}
				if nr_outbits >= 8 {
					nr_outbits -= 8
					if outpos == dstlen {
						return -EFBIG
					}
					vv = outpos
					outpos++
					*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(dst)) + uintptr(vv))) = uint8(outbits >> nr_outbits)
				}
			}
		} else {
			{
				outbits <<= 9
				outbits |= uint32(*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(src)) + uintptr(inpos))))
				nr_outbits += 9

				nr_outbits -= 8
				if outpos == dstlen {
					return -EFBIG
				}
				vv = outpos
				outpos++
				*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(dst)) + uintptr(vv))) = uint8(outbits >> nr_outbits)

				if nr_outbits >= 8 {
					nr_outbits -= 8
					if outpos == dstlen {
						return -EFBIG
					}
					vv = outpos
					outpos++
					*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(dst)) + uintptr(vv))) = uint8(outbits >> nr_outbits)
				}
			}
			{
				outbits <<= 9
				outbits |= uint32(*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(src)) + uintptr(inpos+1))))
				nr_outbits += 9
				nr_outbits -= 8
				if outpos == dstlen {
					return -EFBIG
				}
				vv = outpos
				outpos++
				*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(dst)) + uintptr(vv))) = uint8(outbits >> nr_outbits)
				if nr_outbits >= 8 {
					nr_outbits -= 8
					if outpos == dstlen {
						return -EFBIG
					}
					vv = outpos
					outpos++
					*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(dst)) + uintptr(vv))) = uint8(outbits >> nr_outbits)
				}
			}
		}
	} else if inpos == srclen-1 {
		outbits <<= 9
		outbits |= uint32(*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(src)) + uintptr(inpos))))
		nr_outbits += 9

		nr_outbits -= 8
		if outpos == dstlen {
			return -EFBIG
		}
		vv = outpos
		outpos++
		*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(dst)) + uintptr(vv))) = uint8(outbits >> nr_outbits)

		if nr_outbits >= 8 {
			nr_outbits -= 8
			if outpos == dstlen {
				return -EFBIG
			}
			vv = outpos
			outpos++
			*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(dst)) + uintptr(vv))) = uint8(outbits >> nr_outbits)
		}
	}

	outbits <<= 16
	outbits |= 49152
	nr_outbits += 16
	{
		nr_outbits -= 8
		if outpos == dstlen {
			return -EFBIG
		}
		vv = outpos
		outpos++
		*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(dst)) + uintptr(vv))) = uint8(outbits >> nr_outbits)
	}
	if nr_outbits >= 8 {
		nr_outbits -= 8
		if outpos == dstlen {
			return -EFBIG
		}
		vv = outpos
		outpos++
		*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(dst)) + uintptr(vv))) = uint8(outbits >> nr_outbits)
	}

	return outpos
}

func lzsDecompress(dst *uint8, dstlen int32, src *uint8, srclen int32) int32 {
	var outlen int32 = 0
	var bits_left int32 = 8
	var data uint32
	var offset uint16
	var length uint16
	for {
		{
			if srclen < 2 {
				return -EINVAL
			}
			if 9 >= bits_left {
				data = uint32(int32(*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(src)) + uintptr(0)))) << (9 - bits_left) & 511)
				*(*uintptr)(unsafe.Pointer(&src))++
				srclen--
				bits_left += -1
				if bits_left < 8 {
					data |= uint32(int32(*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(src)) + uintptr(0)))) >> bits_left)
					if !(bits_left != 0) {
						bits_left = 8
						*(*uintptr)(unsafe.Pointer(&src))++
						srclen--
					}
				}
			} else {
				data = uint32(uint64(int32(*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(src)) + uintptr(0))))>>(bits_left-9)) & 511)
				bits_left -= 9
			}
		}
		for data < uint32(256) {
			if outlen == dstlen {
				return -EFBIG
			}

			vv := outlen
			outlen++
			*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(dst)) + uintptr(vv))) = uint8(data)
			{
				if srclen < 2 {
					return -EINVAL
				}
				if 9 >= bits_left {
					data = uint32(int32(*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(src)) + uintptr(0)))) << (9 - bits_left) & 511)
					*(*uintptr)(unsafe.Pointer(&src))++
					srclen--
					bits_left += -1
					if bits_left < 8 {
						data |= uint32(int32(*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(src)) + uintptr(0)))) >> bits_left)
						if !(bits_left != 0) {
							bits_left = 8
							*(*uintptr)(unsafe.Pointer(&src))++
							srclen--
						}
					}
				} else {
					data = uint32(uint64(int32(*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(src)) + uintptr(0))))>>(bits_left-9)) & 511)
					bits_left -= 9
				}
			}
		}
		if data == 384 {
			return outlen
		}
		offset = uint16(data & 127)
		if data < 384 {
			{
				if srclen < 2 {
					return -EINVAL
				}
				if false || 4 >= bits_left {
					data = uint32(int32(*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(src)) + uintptr(0)))) << (4 - bits_left) & 15)
					*(*uintptr)(unsafe.Pointer(&src))++
					srclen--
					bits_left += 4
					if false || bits_left < 8 {
						data |= uint32(int32(*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(src)) + uintptr(0)))) >> bits_left)
						if false && !(bits_left != 0) {
							bits_left = 8
							*(*uintptr)(unsafe.Pointer(&src))++
							srclen--
						}
					}
				} else {
					data = uint32(uint64(int32(*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(src)) + uintptr(0))))>>(bits_left-4)) & 15)
					bits_left -= 4
				}
			}
			offset <<= 4
			offset |= uint16(data)
		}
		{
			if srclen < 2 {
				return -EINVAL
			}
			if false || 2 >= bits_left {
				data = uint32(int32(*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(src)) + uintptr(0)))) << (2 - bits_left) & 3)
				*(*uintptr)(unsafe.Pointer(&src))++
				srclen--
				bits_left += 6
				if false || bits_left < 8 {
					data |= uint32(int32(*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(src)) + uintptr(0)))) >> bits_left)
					if false && !(bits_left != 0) {
						bits_left = 8
						*(*uintptr)(unsafe.Pointer(&src))++
						srclen--
					}
				}
			} else {
				data = uint32(uint64(int32(*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(src)) + uintptr(0))))>>(bits_left-2)) & 3)
				bits_left -= 2
			}
		}
		if data != 3 {
			length = uint16(data + 2)
		} else {
			{
				if srclen < 2 {
					return -EINVAL
				}
				if false || 2 >= bits_left {
					data = uint32(int32(*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(src)) + uintptr(0)))) << (2 - bits_left) & 3)
					*(*uintptr)(unsafe.Pointer(&src))++
					srclen--
					bits_left += 6
					if false || bits_left < 8 {
						data |= uint32(int32(*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(src)) + uintptr(0)))) >> bits_left)
						if false && !(bits_left != 0) {
							bits_left = 8
							*(*uintptr)(unsafe.Pointer(&src))++
							srclen--
						}
					}
				} else {
					data = uint32(uint64(int32(*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(src)) + uintptr(0))))>>(bits_left-2)) & 3)
					bits_left -= 2
				}
			}
			if data != 3 {
				length = uint16(data + 5)
			} else {
				length = uint16(8)
				for 1 != 0 {
					if srclen < 2 {
						return -EINVAL
					}
					if 4 >= bits_left {
						data = uint32(int32(*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(src)) + uintptr(0)))) << (4 - bits_left) & 15)
						*(*uintptr)(unsafe.Pointer(&src))++
						srclen--
						bits_left += 4
						if bits_left < 8 {
							data |= uint32(int32(*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(src)) + uintptr(0)))) >> bits_left)
							if false && !(bits_left != 0) {
								bits_left = 8
								*(*uintptr)(unsafe.Pointer(&src))++
								srclen--
							}
						}
					} else {
						data = uint32(uint64(int32(*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(src)) + uintptr(0))))>>(bits_left-4)) & 15)
						bits_left -= 4
					}

					if data != 15 {
						length += uint16(data)
						break
					}
					length += uint16(15)
				}
			}
		}
		if int32(offset) > outlen {
			return -EINVAL
		}
		if int32(length)+outlen > dstlen {
			return -EFBIG
		}
		for length != 0 {
			*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(dst)) + uintptr(outlen))) = *(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(dst)) + uintptr(outlen-int32(offset))))
			outlen++
			length--
		}
	}
	return -EINVAL
}

func memcmp(s1, s2 uintptr, n uint64) int32 {
	for ; n != 0; n-- {
		c1 := *(*byte)(unsafe.Pointer(s1))
		s1++
		c2 := *(*byte)(unsafe.Pointer(s2))
		s2++
		if c1 < c2 {
			return -1
		}

		if c1 > c2 {
			return 1
		}
	}
	return 0
}
