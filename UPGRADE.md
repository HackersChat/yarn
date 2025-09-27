# Upgrade Notes

The following flags no longer exist:

```
--max-cache-items
--max-cache-ttl
```

Instead use `--max-age-days`, which controls how much of the cache is pulled
back for Timeline, Discover and Mentions views.
