package gocache

import (
	"fmt"
	"log"
	"reflect"
	"testing"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func TestGet(t *testing.T) {
	loadCounts := make(map[string]int, len(db))

	gocache := NewGroup("scores", 2<<10, GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				if _, ok := loadCounts[key]; !ok {
					loadCounts[key] = 0
				}
				loadCounts[key]++
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}),
	)

	for k, v := range db {
		if view, err := gocache.Get(k); err != nil || view.String() != v {
			t.Fatal("failed to get value of Tom")
		} // load from callback function
		if _, err := gocache.Get(k); err != nil || loadCounts[k] > 1 {
			t.Fatalf("cache %s miss", k)
		} // cache hit
	}

	if view, err := gocache.Get("unknown"); err == nil {
		t.Fatalf("the value of unknown should be empty, but %s got", view)
	}

	if _, err := gocache.Get(""); err == nil {
		t.Fatalf(" %s", err)
	}
}

func TestGetter(t *testing.T) {
	// 定义一个函数类型 F，并且实现接口 A 的方法，然后在这个方法中调用自己。
	// 这是 Go 语言中将其他函数（参数返回值定义与 F 一致）转换为接口 A 的常用技巧。
	var f Getter = GetterFunc(func(key string) ([]byte, error) {
		return []byte(key), nil
	})

	expect := []byte("key")
	if v, _ := f.Get("key"); !reflect.DeepEqual(v, expect) {
		t.Errorf("callback failed")
	}
}

func TestGetGroup(t *testing.T) {
	groupName := "scores"
	t.Run("normal", func(t *testing.T) {
		NewGroup(groupName, 2<<10, GetterFunc(
			func(key string) (bytes []byte, err error) { return },
		))
		if group := GetGroup(groupName); group == nil || group.name != groupName {
			t.Fatalf("group %s not exist", groupName)
		}

		if group := GetGroup(groupName + "111"); group != nil {
			t.Fatalf("expect nil, but %s got", group.name)
		}
	})
	t.Run("nil Getter", func(t *testing.T) {
		defer func() { recover() }()
		NewGroup(groupName, 2<<10, nil)
		t.FailNow()
	})
}

func shouldPanic(t *testing.T, f func()) {
	defer func() { recover() }()
	f()
	t.Errorf("should have panicked")
}
