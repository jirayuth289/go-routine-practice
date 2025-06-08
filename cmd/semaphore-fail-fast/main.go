package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
)

type Book struct {
	UserId int
	Id     int
	Title  string
	Body   string
}

func BulkGetBooksWithSemaphore(bookIDs ...string) error {
	const maxConcurrentJobs = 5 // Limit concurrency to 5
	semaphore := make(chan struct{}, maxConcurrentJobs)
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errChan := make(chan error, 1)

	for _, bookID := range bookIDs {
		wg.Add(1)
		semaphore <- struct{}{} // Acquire a slot

		go func(id string) {

			defer wg.Done()
			defer func() { <-semaphore }() // Release the slot

			select {
			case <-ctx.Done():
				return // ไม่ทำอะไร ถ้า context ถูก cancel แล้ว
			default:
			}

			fmt.Printf("Processing job %s\n", id)

			resp, err := GetBook(id)
			if err != nil {
				select {
				case errChan <- fmt.Errorf("------------Job %s Error", id):
					cancel() // ยกเลิก context เมื่อเจอ error
					return
				default:
					return
				}
			}
			fmt.Printf("Job %s done| Title: %s\n", id, resp.Title)
		}(bookID)
	}

	wg.Wait()

	select {
	case err := <-errChan:
		return err
	default:
		return nil
	}
}

func GetBook(id string) (*Book, error) {
	resp, err := http.Get(fmt.Sprintf(`https://jsonplaceholder.typicode.com/posts/%s`, id))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	chunk := []byte{}
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		chunk = append(chunk, scanner.Bytes()...)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	dto := &Book{}
	json.Unmarshal(chunk, dto)

	n, _ := strconv.Atoi(id)
	if n%2 == 0 {
		return dto, nil
	}

	return nil, fmt.Errorf("Error")

}

func main() {
	bookIDs := []string{}

	for i := 1; i <= 10; i++ {
		bookIDs = append(bookIDs, strconv.Itoa(i))
	}

	if err := BulkGetBooksWithSemaphore(bookIDs...); err != nil {
		fmt.Printf("Result: %v", err)
	}
}
