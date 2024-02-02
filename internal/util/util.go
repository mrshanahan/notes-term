package util

import (
    "fmt"
)

func Keys[K comparable, V any](xs map[K]V) []K {
    ks := make([]K, len(xs))
    for k := range xs {
        ks = append(ks, k)
    }
    return ks
}

func MaxBy(xs []string, f func(string)int) *Maybe[string] {
    max := 0
    var maxTmp *Maybe[string] = nil
    for _, x := range xs {
        tmp := f(x)
        if maxTmp == nil || tmp > max {
            max, maxTmp = tmp, Just(x)
        }
    }
    return maxTmp
}

func Max(xs ...int) *Maybe[int] {
    var res *Maybe[int]
    for _, x := range xs {
        if res == nil || x > res.Value {
            res = Just(x)
        }
    }
    return res
}

func Min(xs ...int) *Maybe[int] {
    var res *Maybe[int]
    for _, x := range xs {
        if res == nil || x < res.Value {
            res = Just(x)
        }
    }
    return res
}

type Maybe[T any] struct {
    Value T
}

func Just[T any](x T) *Maybe[T] {
    return &Maybe[T]{x}
}

func LEBytesToUInt32(bs []byte) uint32 {
    i := uint32(bs[3])
    i = i << 8 | uint32(bs[2])
    i = i << 8 | uint32(bs[1])
    i = i << 8 | uint32(bs[0])
    return i
}

func LEBytesToString(bs []byte) string {
    s := "0x"
    for _, b := range bs {
        s = fmt.Sprintf("%s%02x", s, b)
    }
    return s
}
