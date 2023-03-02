# httpgobrrr
Multi thread client to produce synthetic custom HTTP load.
________
[![License](https://img.shields.io/badge/license-MIT-green)](./LICENSE.txt)
[![Release](https://img.shields.io/github/v/tag/sebastianconcept/httpgobrrr?label=release)](https://github.com/sebastianconcept/httpgobrrr/releases)
## Applicability
You can use `httpgobrrr` to perform stress tests on any HTTP service.

## Features
- One producer.
- Multiple consumers.
- One HTTP client per worker.
- Requests customizable via simple JSON file.

## Customize the Requests

In a `jobs/` directory, you can set the request that will be used to hit your target. A zero delayed simple GET /ping for example:

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

If you want a `POST` instead of a `GET`, you can set that using `method` and adding the `payload` and `headers` properties.

## Parametrize quantity and workers

The `ProduceJobs` function has the `count` constant which controls how many times you want `httpgobrrr` to process all the files in the path you give.

```golang
func (producer Producer) ProduceJobs(quantity *int, source_path string) {
	count := *quantity
	for i := 0; i < count; i++ {
		loadJobs(source_path, *producer.inbox)
	}
}
```

## Usage example

     go run main.go -s jobs -c 80 -j 1000

That will search for request definitions to send in the `jobs` path using `80` concurrent threads and producing `1000` requests to be sent.