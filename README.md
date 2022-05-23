# logger

Example 
```go
package main

func main()  {
	myLogger, err := New("logs", "error-2006-01-02.log", 24, 30, 2)
	if err != nil {
		// handle error
	}
	log.SetOutput(myLogger)
}

```