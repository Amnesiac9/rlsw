![Tests](https://github.com/Amnesiac9/rlsw/actions/workflows/tests.yml/badge.svg?branch=main)
![Build](https://github.com/Amnesiac9/rlsw/actions/workflows/build.yml/badge.svg?branch=main)

## Simple sliding window rate limiter package for GO.

The idea behind a sliding window rate limiter is to only allow a set amount of requests in a set block of time. For example, 100 requests within one minute. 
As requests come in, the window gets it's time slots filled up to the limit, and as time passes the window "slides" on a timeline, and the oldest requets fall outside of the window, allowing new requests to come in.

This allows requests to be completed ASAP, without having to worry about going over a specified limit.

Why? I needed a simple, GO routine safe sliding window rate limiter for use in my projects, speicifically when dealing with 3rd party API's with a sliding window rate limit. While there are many other great options for rate limiting, most are overkill for my use case, as I just needed a very simple and lightwait way to schedule requests.

### Usage

Create a rate limiter by calling NewRateLimiter(), with returns a RateLimiterSW.

```
type RateLimiterSW struct {
	mu         sync.Mutex
	timestamps []time.Time
	limit      int
	window     time.Duration
}
```

Call the Wait() function to wait until a timeslot opens up. This will block until the request is allowed. If you need to set a max wait time, you can use `WaitWithLimit(1 * time.Minute)`.

The Wait() function calls a function that returns the time to wait until the request should be allowed, and replaces the oldest timestamp with the time the new request was made. This way, concurrent requests will always be scheduled for the time that the coresponding 

This package expects that the server will attach the rate limiter instance to a specific client or user, so that their rate limits can be cached between requests and kept track of per client. A common wait to do this would be with a `map[string]RateLimiterSW`

The Wait() function automatically removes expired timestamps from the window, so there's no need to worry about removing the timestamps, but if you need to clear the timestamps manually, you can call Clear()
