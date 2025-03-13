# conv

Golang library for safe base type conversion and struct/map manipulation.

## How to use

```golang
package main

import (
    "internal/conv"
    "fmt"
    "reflect"
)

func main() {
    a, b, c := "8", 1, 1.04
    a2, b2, c2 := conv.Rune(a), conv.Float64(b), conv.String(c)

    fmt.Println(a2, b2, c2)
    fmt.Println(reflect.TypeOf(a2), reflect.TypeOf(b2), reflect.TypeOf(c2))
}

// out :
// 8 1 1.04
// int32 float64 string
```
