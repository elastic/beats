package goracle

import "os"
import "fmt"

func init() {
	fmt.Println("Goracle is deprecated because of naming (trademark) issues. Please use github.com/godror/godror instead!")
	fmt.Fprintln(os.Stderr, "Goracle is deprecated because of naming (trademark) issues. Please use github.com/godror/godror instead!")
}
