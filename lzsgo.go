package lzsgo

import (
	"errors"
	"math/bits"
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

type hashTable struct {
	entries    [htSize]uint32
	generation uint32
}

var hashTablePool = sync.Pool{
	New: func() interface{} {
		return &hashTable{}
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

func lzsCompress(dst *uint8, dstlen int32, src *uint8, srclen int32) int32 {
	ht := hashTablePool.Get().(*hashTable)
	result := lzsCompressCore(dst, dstlen, src, srclen, ht)
	hashTablePool.Put(ht)
	return result
}

func lzsCompressCore(dst *uint8, dstlen int32, src *uint8, srclen int32, ht *hashTable) int32 {
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

	table := &ht.entries
	entries := table[:]
	gen := (ht.generation + 1) & 0xFFFF
	if gen == 0 {
		for i := range entries {
			entries[i] = 0
		}
		gen = 1
	}
	ht.generation = gen
	currentGen := gen

	// Pre-calculate pointer base addresses
	srcPtr := uintptr(unsafe.Pointer(src))
	dstPtr := uintptr(unsafe.Pointer(dst))

	if srclen > htSize {
		return -EFBIG
	}

	for inpos < srclen-2 {
		hash = *(*uint16)(unsafe.Pointer((*uint8)(unsafe.Pointer(srcPtr + uintptr(inpos)))))
		idx := int(hash)
		entry := entries[idx]
		hofs = 65535
		if entry>>16 == currentGen {
			hofs = uint16(entry)
		}

		hash_chain[inpos&2047] = hofs
		entries[idx] = (currentGen << 16) | uint32(uint16(inpos))

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
			idx := int(hash)
			hofs_tmp := uint16(65535)
			entry := entries[idx]
			if entry>>16 == currentGen {
				hofs_tmp = uint16(entry)
			}
			hash_chain[inpos&2047] = hofs_tmp

			entries[idx] = (currentGen << 16) | uint32(uint16(inpos))

			inpos++
			longest_match_len--
		}
	}

	// Handle remaining 1-2 bytes
	if inpos == srclen-2 {
		hash = *(*uint16)(unsafe.Pointer((*uint8)(unsafe.Pointer(srcPtr + uintptr(inpos)))))
		idx := int(hash)
		hofs = 65535
		entry := entries[idx]
		if entry>>16 == currentGen {
			hofs = uint16(entry)
		}
		entries[idx] = (currentGen << 16) | uint32(uint16(inpos))
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
	var data uint32
	var offset uint16
	var length uint16

	// Pre-calculate pointer base addresses
	srcPtr := uintptr(unsafe.Pointer(src))
	dstPtr := uintptr(unsafe.Pointer(dst))

	// Bit reading optimization - read multiple bytes ahead
	var bitBuf uint64 = 0
	var bitCount uint32 = 0
	var inpos int32 = 0

	for {
		// Ensure we have at least 9 bits available
		if bitCount < 9 {
			if inpos+8 <= srclen {
				// Fast path: read 32 bits at once
				val32 := *(*uint32)(unsafe.Pointer(srcPtr + uintptr(inpos)))
				val32 = bits.ReverseBytes32(val32)
				bitBuf = (bitBuf << 32) | uint64(val32)
				bitCount += 32
				inpos += 4
			} else {
				for bitCount < 9 && inpos < srclen {
					bitBuf = (bitBuf << 8) | uint64(*(*uint8)(unsafe.Pointer(srcPtr + uintptr(inpos))))
					inpos++
					bitCount += 8
				}
				if bitCount < 9 {
					return -EINVAL
				}
			}
		}

		// Read 9-bit literal or offset marker
		data = uint32((bitBuf >> (bitCount - 9)) & 511)
		bitCount -= 9

		for data < 256 {
			if outlen == dstlen {
				return -EFBIG
			}

			*(*uint8)(unsafe.Pointer(dstPtr + uintptr(outlen))) = uint8(data)
			outlen++

			// Read next 9 bits
			if bitCount < 9 {
				if inpos+8 <= srclen {
					val32 := *(*uint32)(unsafe.Pointer(srcPtr + uintptr(inpos)))
					val32 = bits.ReverseBytes32(val32)
					bitBuf = (bitBuf << 32) | uint64(val32)
					bitCount += 32
					inpos += 4
				} else {
					for bitCount < 9 && inpos < srclen {
						bitBuf = (bitBuf << 8) | uint64(*(*uint8)(unsafe.Pointer(srcPtr + uintptr(inpos))))
						inpos++
						bitCount += 8
					}
					if bitCount < 9 {
						return -EINVAL
					}
				}
			}
			data = uint32((bitBuf >> (bitCount - 9)) & 511)
			bitCount -= 9
		}
		if data == 384 {
			return outlen
		}
		offset = uint16(data & 127)
		if data < 384 {
			// Read 4 more bits for extended offset
			if bitCount < 4 {
				if inpos+8 <= srclen {
					val32 := *(*uint32)(unsafe.Pointer(srcPtr + uintptr(inpos)))
					val32 = bits.ReverseBytes32(val32)
					bitBuf = (bitBuf << 32) | uint64(val32)
					bitCount += 32
					inpos += 4
				} else {
					for bitCount < 4 && inpos < srclen {
						bitBuf = (bitBuf << 8) | uint64(*(*uint8)(unsafe.Pointer(srcPtr + uintptr(inpos))))
						inpos++
						bitCount += 8
					}
					if bitCount < 4 {
						return -EINVAL
					}
				}
			}
			offset = (offset << 4) | uint16((bitBuf>>(bitCount-4))&15)
			bitCount -= 4
		}

		// Read 2 bits for length code
		if bitCount < 2 {
			if inpos+8 <= srclen {
				val32 := *(*uint32)(unsafe.Pointer(srcPtr + uintptr(inpos)))
				val32 = bits.ReverseBytes32(val32)
				bitBuf = (bitBuf << 32) | uint64(val32)
				bitCount += 32
				inpos += 4
			} else {
				for bitCount < 2 && inpos < srclen {
					bitBuf = (bitBuf << 8) | uint64(*(*uint8)(unsafe.Pointer(srcPtr + uintptr(inpos))))
					inpos++
					bitCount += 8
				}
				if bitCount < 2 {
					return -EINVAL
				}
			}
		}
		data = uint32((bitBuf >> (bitCount - 2)) & 3)
		bitCount -= 2

		if data != 3 {
			length = uint16(data + 2)
		} else {
			// Read another 2 bits
			if bitCount < 2 {
				if inpos+8 <= srclen {
					val32 := *(*uint32)(unsafe.Pointer(srcPtr + uintptr(inpos)))
					val32 = bits.ReverseBytes32(val32)
					bitBuf = (bitBuf << 32) | uint64(val32)
					bitCount += 32
					inpos += 4
				} else {
					for bitCount < 2 && inpos < srclen {
						bitBuf = (bitBuf << 8) | uint64(*(*uint8)(unsafe.Pointer(srcPtr + uintptr(inpos))))
						inpos++
						bitCount += 8
					}
					if bitCount < 2 {
						return -EINVAL
					}
				}
			}
			data = uint32((bitBuf >> (bitCount - 2)) & 3)
			bitCount -= 2

			if data != 3 {
				length = uint16(data + 5)
			} else {
				length = uint16(8)
				for {
					if bitCount < 4 {
						if inpos+8 <= srclen {
							val32 := *(*uint32)(unsafe.Pointer(srcPtr + uintptr(inpos)))
							val32 = bits.ReverseBytes32(val32)
							bitBuf = (bitBuf << 32) | uint64(val32)
							bitCount += 32
							inpos += 4
						} else {
							for bitCount < 4 && inpos < srclen {
								bitBuf = (bitBuf << 8) | uint64(*(*uint8)(unsafe.Pointer(srcPtr + uintptr(inpos))))
								inpos++
								bitCount += 8
							}
							if bitCount < 4 {
								return -EINVAL
							}
						}
					}
					data = uint32((bitBuf >> (bitCount - 4)) & 15)
					bitCount -= 4

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

		// Copy matched data
		copyOffset := outlen - int32(offset)
		copyLen := int32(length)

		// Optimization: batch copy when offset >= length (non-overlapping)
		if int32(offset) >= copyLen {
			// Use word-aligned copies for better performance
			srcPos := dstPtr + uintptr(copyOffset)
			dstPos := dstPtr + uintptr(outlen)

			// Copy 8 bytes at a time
			for copyLen >= 8 {
				*(*uint64)(unsafe.Pointer(dstPos)) = *(*uint64)(unsafe.Pointer(srcPos))
				srcPos += 8
				dstPos += 8
				copyLen -= 8
			}
			// Copy 4 bytes if possible
			if copyLen >= 4 {
				*(*uint32)(unsafe.Pointer(dstPos)) = *(*uint32)(unsafe.Pointer(srcPos))
				srcPos += 4
				dstPos += 4
				copyLen -= 4
			}
			// Copy remaining bytes
			for copyLen > 0 {
				*(*uint8)(unsafe.Pointer(dstPos)) = *(*uint8)(unsafe.Pointer(srcPos))
				srcPos++
				dstPos++
				copyLen--
			}
			outlen += int32(length)
		} else {
			// Overlapping copy - must go byte by byte for RLE patterns
			for length != 0 {
				*(*uint8)(unsafe.Pointer(dstPtr + uintptr(outlen))) = *(*uint8)(unsafe.Pointer(dstPtr + uintptr(copyOffset)))
				outlen++
				copyOffset++
				length--
			}
		}
	}
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

func memcmp(s1, s2 unsafe.Pointer, n uintptr) int {
	if n == 0 {
		return 0
	}

	ptr1 := uintptr(s1)
	ptr2 := uintptr(s2)

	for n >= 8 {
		v1 := *(*uint64)(unsafe.Pointer(ptr1))
		v2 := *(*uint64)(unsafe.Pointer(ptr2))
		if v1 != v2 {
			diff := v1 ^ v2
			shift := uint(bits.TrailingZeros64(diff) &^ 7)
			b1 := byte(v1 >> shift)
			b2 := byte(v2 >> shift)
			if b1 < b2 {
				return -1
			}
			return 1
		}
		ptr1 += 8
		ptr2 += 8
		n -= 8
	}

	for n > 0 {
		b1 := *(*uint8)(unsafe.Pointer(ptr1))
		b2 := *(*uint8)(unsafe.Pointer(ptr2))
		if b1 < b2 {
			return -1
		} else if b1 > b2 {
			return 1
		}
		ptr1++
		ptr2++
		n--
	}

	return 0
}
