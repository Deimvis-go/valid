package hi

import "fmt"

func Main() {
	s := MyStruct{
		Answer: 99,
	}
	err := Validate(s)
	fmt.Println(err) // MyStruct: answer is wrong
}
