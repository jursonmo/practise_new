package main

import (
	"fmt"
	"reflect"

	"github.com/jursonmo/practise_new/pkg/hash"
)

func main() {
	ch := hash.NewConsistentHash()
	for i := 0; i < 3; i++ {
		ch.AddWithWeight(fmt.Sprintf("node-%d", i), 10)
	}

	//查找10个key对应的node。
	findNodes := func() []any {
		values := make([]any, 0, 10)
		for i := 0; i < 10; i++ {
			v, ok := ch.Get(fmt.Sprintf("key-%d", i))
			if !ok {
				panic("not found")
			}
			fmt.Println(v)
			values = append(values, v)
		}
		return values
	}
	nodes1 := findNodes()

	//删除node-0，看结果是否跟之前的一样。
	fmt.Println("remove node-0")
	ch.Remove("node-0")
	nodes2 := findNodes()
	if reflect.DeepEqual(nodes1, nodes2) {
		fmt.Println("nodes1 and nodes2 are equal")
		panic("nodes1 and nodes2 are equal")
	} else {
		fmt.Println("nodes1 and nodes2 are not equal")
	}

	//把node-0加回来，看结果是否跟之前的一样。证明了一致性hash的结果是稳定的。
	fmt.Println("add node-0 back, check the result if it is the same as before")
	ch.AddWithWeight("node-0", 10)
	nodes3 := findNodes()
	//比较nodes1和nodes3是否相等.相同说明是稳定的。
	if reflect.DeepEqual(nodes1, nodes3) {
		fmt.Println("nodes1 and nodes3 are equal, it is the same as before, it is stable")
	} else {
		fmt.Println("nodes1 and nodes3 are not equal")
		panic("nodes1 and nodes3 are not equal")
	}
}
