package taintflow

import (
	"fmt"
	"maps"
	"strings"
	"testing"
)

func scanWildcardWrites(store state, key cell, item value) value {
	for candidate, stored := range store.Memory {
		if candidate.Object != key.Object {
			continue
		}
		fields := candidate.Fields
		if fields != "*" && !strings.HasSuffix(fields, "/*") {
			continue
		}
		prefix := strings.TrimSuffix(fields, "/*")
		if fields == "*" || prefix == "" || prefix == key.Fields || strings.HasPrefix(key.Fields, prefix+"/") {
			item = joinValue(item, stored)
		}
	}
	return item
}

func wildcardTestPaths() []string {
	paths := []string{"", "0/1/2", "0/10", "é/δ", "capture:1", "/*"}
	level := []string{""}
	for range 3 {
		var next []string
		for _, prefix := range level {
			for _, suffix := range []string{"0", "1", "/", "*"} {
				next = append(next, prefix+suffix)
			}
		}
		paths = append(paths, next...)
		level = next
	}
	return paths
}

func wildcardTestValue(index int) value {
	item := emptyValue()
	item.Data[atom{Source: Site{Function: fmt.Sprint(index)}}] = true
	item.Objects[object{Allocation: fmt.Sprint(index)}] = true
	item.Functions[fmt.Sprint(index)] = true
	item.Nil, item.Opaque = index%2 == 0, index%3 == 0
	return item
}

func TestWildcardLookupMatchesFullScan(t *testing.T) {
	paths := wildcardTestPaths()
	roots := []object{{Input: 1}, {Input: 2}, {Input: 1, Fields: "0"}, {Input: 1, Captured: true}, {Allocation: "local"}}
	store := emptyState()
	for i, root := range roots {
		for j, path := range paths {
			item := wildcardTestValue(i*len(paths) + j)
			store.Memory[cell{Object: root, Fields: path}] = item
			store.Memory[cell{Object: root, Fields: path + "/*"}] = item
		}
	}
	for _, root := range append(roots, object{Input: 3}) {
		for _, path := range paths {
			key := cell{Object: root, Fields: path}
			initial := wildcardTestValue(-1)
			want := scanWildcardWrites(store, key, initial)
			if got := wildcardWrites(store, key, initial); !valueEqual(got, want) {
				t.Fatalf("root=%+v fields=%q: wildcard dependencies differ", root, path)
			}
		}
	}
	if got := wildcardWrites(emptyState(), cell{}, value{}); !valueEqual(got, value{}) {
		t.Fatal("empty memory adds dependencies")
	}
}

func TestWildcardLookupPreservesStoredValues(t *testing.T) {
	store := emptyState()
	root := object{Input: 1}
	before := emptyState()
	for i, path := range []string{"*", "/*", "0/*", "0/1/*", "0/1/2/*"} {
		key := cell{Object: root, Fields: path}
		item := wildcardTestValue(i)
		store.Memory[key] = item
		item.Data = maps.Clone(item.Data)
		item.Objects = maps.Clone(item.Objects)
		item.Functions = maps.Clone(item.Functions)
		before.Memory[key] = item
	}
	initial := wildcardTestValue(-1)
	got := wildcardWrites(store, cell{Object: root, Fields: "0/1/2"}, initial)
	if len(got.Data) != 6 || !stateEqual(store, before) || len(initial.Data) != 1 {
		t.Fatal("wildcard union loses facts or changes its inputs")
	}
}

func BenchmarkWildcardWrites(b *testing.B) {
	for _, size := range []int{16, 256, 4096} {
		for _, wildcard := range []bool{false, true} {
			b.Run(fmt.Sprintf("cells=%d/wildcard=%t", size, wildcard), func(b *testing.B) {
				store := emptyState()
				key := cell{Object: object{Input: 1}, Fields: "0/1/2"}
				for i := range size {
					store.Memory[cell{Object: key.Object, Fields: fmt.Sprintf("other/%d", i)}] = value{}
				}
				if wildcard {
					store.Memory[cell{Object: key.Object, Fields: "0/*"}] = wildcardTestValue(1)
				}
				for _, method := range []struct {
					name string
					read func(state, cell, value) value
				}{{"scan", scanWildcardWrites}, {"lookup", wildcardWrites}} {
					b.Run(method.name, func(b *testing.B) {
						b.ReportAllocs()
						for b.Loop() {
							got := method.read(store, key, value{})
							if wildcard && len(got.Data) != 1 {
								b.Fatal("lost wildcard dependency")
							}
						}
					})
				}
			})
		}
	}
}
