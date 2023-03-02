# httpgobrrr
Multi thread client to produce synthetic custom HTTP load.
________
[![Release](https://img.shields.io/github/v/tag/sebastianconcept/Mapless?label=release)](https://github.com/sebastianconcept/Mapless/releases)

## Applicability
You can use `httpgobrrr` to perform stress tests on any HTTP service.

## Features
- One producer.
- Multiple consumers.
- One HTTP client per worker.
- Requests customizable via simple JSON file.

## Customize the Requests

In the `jobs/` directory, you can set the request that will be used to hit your target. A zero delayed simple GET /ping for example:

```JSON
{
  "delay": 0,
  "url": "http://127.0.0.1:1701/ping",
  "method": "GET",
  "headers": {},
  "payload": {}
}
```
*`delay` is in milliseconds.
## Parametrize quantity and workers

The `ProduceJobs` function has the `count` constant which controls how many times you want `bearer` to process all the files under the `jobs/` directory.

```golang
func (producer Producer) ProduceJobs() {
	count := 1000
	for i := 0; i < count; i++ {
		loadJobs("jobs", *producer.inbox)
	}
}
```

And the `main` function can configure the `quantityOfWorkers` constant which will define how many requests will be sending at the same time (threads with an http client each one).

```golang
func main() {
	inbox := make(chan Job, 1000)
	const quantityOfWorkers = 100
  ```

## Run

    go run .
