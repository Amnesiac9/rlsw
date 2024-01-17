## Simple sliding window rate limiter package for GO.

The idea behind a sliding window rate limiter is to only allow a set amount of requests in a set block of time. For example, 100 requests within one minute. 
As requests come in, the window gets filled up to the limit, and the next time a request is made, if no requests have moved outside of the window - i.e become older than a minute - then the request is denied. 

You can use the Wait() function to instead wait until a timeslot opens up. This will block until the request is allowed.

Currently, not meant for use for concurrency accesing the same ratelimiter in a loop. This will cause requests that come in to all wait for the same amount of time, and fire all at once. Planned support for concurrency coming soon.