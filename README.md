# Go Util

A repo for my Golang util libraries.

## sync/successgroup
This package is the counter-part of `golang.org/x/sync/errgroup`.

Instead of returning when an error occurs, it instead returns
**immediately** when there's a successful result. This can be used to issue
the same request to several different sources and just use the first result.

### Example

The following example will fire the same request to two different databases.
This is an example of improved latency (`min(dbPrimary, dbReadReplica)`)
and improved availability (`dbPrimaryHealthy || dbReadReplicaHealthy`).

```go
group, ctx := successgroup.WithContext(context.Background())

for db := range []Database{dbPrimary, dbReadReplica} {
    group.Go(func() error {
        return db.GetUsersCount(ctx)
    })
}
userCountRaw, dbErr := group.Wait()
if dbErr != nil {
    panic(fmt.Sprintf("Database error: %s", dbErr.Error()))
}
userCount := userCountRaw.(int)
fmt.Println("Got user count:", userCount)
```
