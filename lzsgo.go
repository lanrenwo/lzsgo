package lzsgo

import (
	"errors"
	"sync"
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

var hashTablePool = sync.Pool{
	New: func() interface{} {
		return new([htSize]uint16)
	},
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
	hash_table := hashTablePool.Get().(*[htSize]uint16)
	result := lzsCompressCore(dst, dstlen, src, srclen, hash_table)
	hashTablePool.Put(hash_table)
	return result
}

func lzsCompressCore(dst *uint8, dstlen int32, src *uint8, srclen int32, hash_table *[htSize]uint16) int32 {
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
	var hash_chain [2048]uint16
	var vv int32

	// Bitmap for lazy hash table initialization (1KB vs 128KB copy)
	var valid_bitmap [htSize >> 6]uint64

	// Pre-calculate pointer base addresses
	srcPtr := uintptr(unsafe.Pointer(src))
	dstPtr := uintptr(unsafe.Pointer(dst))

	if srclen > htSize {
		return -EFBIG
	}

	for inpos < srclen-2 {
		hash = *(*uint16)(unsafe.Pointer((*uint8)(unsafe.Pointer(srcPtr + uintptr(inpos)))))

		// Check if hash slot is initialized via bitmap
		bit_idx := hash >> 6
		bit_pos := hash & 63
		hofs = 65535
		if (valid_bitmap[bit_idx] & (1 << bit_pos)) != 0 {
			hofs = hash_table[hash]
		}

		hash_chain[inpos&2047] = hofs
		hash_table[hash] = uint16(inpos)
		valid_bitmap[bit_idx] |= (1 << bit_pos)

		// Literal encoding (no match found or out of window)
		if int32(hofs)+2048 <= inpos || int32(hofs) == 65535 {
			outbits <<= 9
			outbits |= uint32(*(*uint8)(unsafe.Pointer(srcPtr + uintptr(inpos))))
			nr_outbits += 9
			{
				nr_outbits -= 8
				if outpos == dstlen {
					return -EFBIG
				}
				vv = outpos
				outpos++
				*(*uint8)(unsafe.Pointer(dstPtr + uintptr(vv))) = uint8(outbits >> nr_outbits)
			}
			if nr_outbits >= 8 {
				nr_outbits -= 8
				if outpos == dstlen {
					return -EFBIG
				}
				vv = outpos
				outpos++
				*(*uint8)(unsafe.Pointer(dstPtr + uintptr(vv))) = uint8(outbits >> nr_outbits)
			}
			inpos++
			continue
		}

		longest_match_len = 2
		longest_match_ofs = hofs
		for ; int32(hofs) != 65535 && int32(hofs)+2048 > inpos; hofs = hash_chain[hofs&2047] {
			if !(memcmp(unsafe.Pointer(srcPtr+uintptr(hofs+2)), unsafe.Pointer(srcPtr+uintptr(inpos+2)), uintptr(longest_match_len-1)) != 0) {
				longest_match_ofs = hofs
				for {
					longest_match_len++
					if int32(longest_match_len)+inpos == srclen {
						goto got_match
					}
					if !(int32(*(*uint8)(unsafe.Pointer(srcPtr + uintptr(int32(longest_match_len)+inpos)))) == int32(*(*uint8)(unsafe.Pointer(srcPtr + uintptr(int32(longest_match_len)+int32(hofs)))))) {
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
			*(*uint8)(unsafe.Pointer(dstPtr + uintptr(vv))) = uint8(outbits >> nr_outbits)

			if nr_outbits >= 8 {
				nr_outbits -= 8
				if outpos == dstlen {
					return -EFBIG
				}
				vv = outpos
				outpos++
				*(*uint8)(unsafe.Pointer(dstPtr + uintptr(vv))) = uint8(outbits >> nr_outbits)
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
			*(*uint8)(unsafe.Pointer(dstPtr + uintptr(vv))) = uint8(outbits >> nr_outbits)
			if nr_outbits >= 8 {
				nr_outbits -= 8
				if outpos == dstlen {
					return -EFBIG
				}
				vv = outpos
				outpos++
				*(*uint8)(unsafe.Pointer(dstPtr + uintptr(vv))) = uint8(outbits >> nr_outbits)
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
				*(*uint8)(unsafe.Pointer(dstPtr + uintptr(vv))) = uint8(outbits >> nr_outbits)
			}
			if nr_outbits >= 8 {
				nr_outbits -= 8
				if outpos == dstlen {
					return -EFBIG
				}
				vv = outpos
				outpos++
				*(*uint8)(unsafe.Pointer(dstPtr + uintptr(vv))) = uint8(outbits >> nr_outbits)
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
				*(*uint8)(unsafe.Pointer(dstPtr + uintptr(vv))) = uint8(outbits >> nr_outbits)
			}
			if nr_outbits >= 8 {
				nr_outbits -= 8
				if outpos == dstlen {
					return -EFBIG
				}
				vv = outpos
				outpos++
				*(*uint8)(unsafe.Pointer(dstPtr + uintptr(vv))) = uint8(outbits >> nr_outbits)
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
					*(*uint8)(unsafe.Pointer(dstPtr + uintptr(vv))) = uint8(outbits >> nr_outbits)
				}
				if nr_outbits >= 8 {
					nr_outbits -= 8
					if outpos == dstlen {
						return -EFBIG
					}
					vv = outpos
					outpos++
					*(*uint8)(unsafe.Pointer(dstPtr + uintptr(vv))) = uint8(outbits >> nr_outbits)
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
					*(*uint8)(unsafe.Pointer(dstPtr + uintptr(vv))) = uint8(outbits >> nr_outbits)
				}
				if nr_outbits >= 8 {
					nr_outbits -= 8
					if outpos == dstlen {
						return -EFBIG
					}
					vv = outpos
					outpos++
					*(*uint8)(unsafe.Pointer(dstPtr + uintptr(vv))) = uint8(outbits >> nr_outbits)
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
					*(*uint8)(unsafe.Pointer(dstPtr + uintptr(vv))) = uint8(outbits >> nr_outbits)
				}
				if nr_outbits >= 8 {
					nr_outbits -= 8
					if outpos == dstlen {
						return -EFBIG
					}
					vv = outpos
					outpos++
					*(*uint8)(unsafe.Pointer(dstPtr + uintptr(vv))) = uint8(outbits >> nr_outbits)
				}
			}
		}
		if inpos+int32(longest_match_len) >= srclen-2 {
			inpos += int32(longest_match_len)
			break
		}
		inpos++
		longest_match_len--
		for longest_match_len != 0 {
			hash = *(*uint16)(unsafe.Pointer((*uint8)(unsafe.Pointer(srcPtr + uintptr(inpos)))))

			bit_idx := hash >> 6
			bit_pos := hash & 63
			hofs_tmp := uint16(65535)
			if (valid_bitmap[bit_idx] & (1 << bit_pos)) != 0 {
				hofs_tmp = hash_table[hash]
			}
			hash_chain[inpos&2047] = hofs_tmp

			hash_table[hash] = uint16(inpos)
			valid_bitmap[bit_idx] |= (1 << bit_pos)

			inpos++
			longest_match_len--
		}
	}

	// Handle remaining 1-2 bytes
	if inpos == srclen-2 {
		hash = *(*uint16)(unsafe.Pointer((*uint8)(unsafe.Pointer(srcPtr + uintptr(inpos)))))

		bit_idx := hash >> 6
		bit_pos := hash & 63
		hofs = 65535
		if (valid_bitmap[bit_idx] & (1 << bit_pos)) != 0 {
			hofs = hash_table[hash]
		}
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
					*(*uint8)(unsafe.Pointer(dstPtr + uintptr(vv))) = uint8(outbits >> nr_outbits)
					if nr_outbits >= 8 {
						nr_outbits -= 8
						if outpos == dstlen {
							return -EFBIG
						}
						vv = outpos
						outpos++
						*(*uint8)(unsafe.Pointer(dstPtr + uintptr(vv))) = uint8(outbits >> nr_outbits)
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
					*(*uint8)(unsafe.Pointer(dstPtr + uintptr(vv))) = uint8(outbits >> nr_outbits)
					if nr_outbits >= 8 {
						nr_outbits -= 8
						if outpos == dstlen {
							return -EFBIG
						}
						vv = outpos
						outpos++
						*(*uint8)(unsafe.Pointer(dstPtr + uintptr(vv))) = uint8(outbits >> nr_outbits)
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
					*(*uint8)(unsafe.Pointer(dstPtr + uintptr(vv))) = uint8(outbits >> nr_outbits)
				}
				if nr_outbits >= 8 {
					nr_outbits -= 8
					if outpos == dstlen {
						return -EFBIG
					}
					vv = outpos
					outpos++
					*(*uint8)(unsafe.Pointer(dstPtr + uintptr(vv))) = uint8(outbits >> nr_outbits)
				}
			}
		} else {
			{
				outbits <<= 9
				outbits |= uint32(*(*uint8)(unsafe.Pointer(srcPtr + uintptr(inpos))))
				nr_outbits += 9

				nr_outbits -= 8
				if outpos == dstlen {
					return -EFBIG
				}
				vv = outpos
				outpos++
				*(*uint8)(unsafe.Pointer(dstPtr + uintptr(vv))) = uint8(outbits >> nr_outbits)

				if nr_outbits >= 8 {
					nr_outbits -= 8
					if outpos == dstlen {
						return -EFBIG
					}
					vv = outpos
					outpos++
					*(*uint8)(unsafe.Pointer(dstPtr + uintptr(vv))) = uint8(outbits >> nr_outbits)
				}
			}
			{
				outbits <<= 9
				outbits |= uint32(*(*uint8)(unsafe.Pointer(srcPtr + uintptr(inpos+1))))
				nr_outbits += 9
				nr_outbits -= 8
				if outpos == dstlen {
					return -EFBIG
				}
				vv = outpos
				outpos++
				*(*uint8)(unsafe.Pointer(dstPtr + uintptr(vv))) = uint8(outbits >> nr_outbits)
				if nr_outbits >= 8 {
					nr_outbits -= 8
					if outpos == dstlen {
						return -EFBIG
					}
					vv = outpos
					outpos++
					*(*uint8)(unsafe.Pointer(dstPtr + uintptr(vv))) = uint8(outbits >> nr_outbits)
				}
			}
		}
	} else if inpos == srclen-1 {
		outbits <<= 9
		outbits |= uint32(*(*uint8)(unsafe.Pointer(srcPtr + uintptr(inpos))))
		nr_outbits += 9

		nr_outbits -= 8
		if outpos == dstlen {
			return -EFBIG
		}
		vv = outpos
		outpos++
		*(*uint8)(unsafe.Pointer(dstPtr + uintptr(vv))) = uint8(outbits >> nr_outbits)

		if nr_outbits >= 8 {
			nr_outbits -= 8
			if outpos == dstlen {
				return -EFBIG
			}
			vv = outpos
			outpos++
			*(*uint8)(unsafe.Pointer(dstPtr + uintptr(vv))) = uint8(outbits >> nr_outbits)
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
		*(*uint8)(unsafe.Pointer(dstPtr + uintptr(vv))) = uint8(outbits >> nr_outbits)
	}
	if nr_outbits >= 8 {
		nr_outbits -= 8
		if outpos == dstlen {
			return -EFBIG
		}
		vv = outpos
		outpos++
		*(*uint8)(unsafe.Pointer(dstPtr + uintptr(vv))) = uint8(outbits >> nr_outbits)
	}

	return outpos
}

func lzsDecompress(dst *uint8, dstlen int32, src *uint8, srclen int32) int32 {
	var outlen int32 = 0
	var bits_left int32 = 8
	var data uint32
	var offset uint16
	var length uint16

	// Pre-calculate pointer base addresses
	srcPtr := uintptr(unsafe.Pointer(src))
	dstPtr := uintptr(unsafe.Pointer(dst))

	for {
		// Read 9-bit literal or offset marker
		{
			if srclen < 2 {
				return -EINVAL
			}
			if 9 >= bits_left {
				data = uint32(int32(*(*uint8)(unsafe.Pointer(srcPtr)))<<(9-bits_left)) & 511
				srcPtr++
				srclen--
				bits_left += -1
				if bits_left < 8 {
					data |= uint32(int32(*(*uint8)(unsafe.Pointer(srcPtr))) >> bits_left)
					if !(bits_left != 0) {
						bits_left = 8
						srcPtr++
						srclen--
					}
				}
			} else {
				data = uint32(int32(*(*uint8)(unsafe.Pointer(srcPtr)))>>(bits_left-9)) & 511
				bits_left -= 9
			}
		}
		for data < uint32(256) {
			if outlen == dstlen {
				return -EFBIG
			}

			*(*uint8)(unsafe.Pointer(dstPtr + uintptr(outlen))) = uint8(data)
			outlen++

			{
				if srclen < 2 {
					return -EINVAL
				}
				if 9 >= bits_left {
					data = uint32(int32(*(*uint8)(unsafe.Pointer(srcPtr)))<<(9-bits_left)) & 511
					srcPtr++
					srclen--
					bits_left += -1
					if bits_left < 8 {
						data |= uint32(int32(*(*uint8)(unsafe.Pointer(srcPtr))) >> bits_left)
						if !(bits_left != 0) {
							bits_left = 8
							srcPtr++
							srclen--
						}
					}
				} else {
					data = uint32(int32(*(*uint8)(unsafe.Pointer(srcPtr)))>>(bits_left-9)) & 511
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
				if 4 >= bits_left {
					data = uint32(int32(*(*uint8)(unsafe.Pointer(srcPtr)))<<(4-bits_left)) & 15
					srcPtr++
					srclen--
					bits_left += 4
					if bits_left < 8 {
						data |= uint32(int32(*(*uint8)(unsafe.Pointer(srcPtr))) >> bits_left)
					}
				} else {
					data = uint32(int32(*(*uint8)(unsafe.Pointer(srcPtr)))>>(bits_left-4)) & 15
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
			if 2 >= bits_left {
				data = uint32(int32(*(*uint8)(unsafe.Pointer(srcPtr)))<<(2-bits_left)) & 3
				srcPtr++
				srclen--
				bits_left += 6
				if bits_left < 8 {
					data |= uint32(int32(*(*uint8)(unsafe.Pointer(srcPtr))) >> bits_left)
				}
			} else {
				data = uint32(int32(*(*uint8)(unsafe.Pointer(srcPtr)))>>(bits_left-2)) & 3
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
				if 2 >= bits_left {
					data = uint32(int32(*(*uint8)(unsafe.Pointer(srcPtr)))<<(2-bits_left)) & 3
					srcPtr++
					srclen--
					bits_left += 6
					if bits_left < 8 {
						data |= uint32(int32(*(*uint8)(unsafe.Pointer(srcPtr))) >> bits_left)
					}
				} else {
					data = uint32(int32(*(*uint8)(unsafe.Pointer(srcPtr)))>>(bits_left-2)) & 3
					bits_left -= 2
				}
			}
			if data != 3 {
				length = uint16(data + 5)
			} else {
				length = uint16(8)
				for {
					if srclen < 2 {
						return -EINVAL
					}
					if 4 >= bits_left {
						data = uint32(int32(*(*uint8)(unsafe.Pointer(srcPtr)))<<(4-bits_left)) & 15
						srcPtr++
						srclen--
						bits_left += 4
						if bits_left < 8 {
							data |= uint32(int32(*(*uint8)(unsafe.Pointer(srcPtr))) >> bits_left)
						}
					} else {
						data = uint32(int32(*(*uint8)(unsafe.Pointer(srcPtr)))>>(bits_left-4)) & 15
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

		// Copy matched data - handle overlapping case byte by byte
		copyOffset := outlen - int32(offset)
		for length != 0 {
			*(*uint8)(unsafe.Pointer(dstPtr + uintptr(outlen))) = *(*uint8)(unsafe.Pointer(dstPtr + uintptr(copyOffset)))
			outlen++
			copyOffset++
			length--
		}
	}
	return -EINVAL
}

func memcmp(s1, s2 unsafe.Pointer, n uintptr) int {
	p1 := (*[1 << 30]byte)(s1)[:n:n]
	p2 := (*[1 << 30]byte)(s2)[:n:n]
	for i := 0; i < len(p1); i++ {
		if p1[i] < p2[i] {
			return -1
		} else if p1[i] > p2[i] {
			return 1
		}
	}
	return 0
}
