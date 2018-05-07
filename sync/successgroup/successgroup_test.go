package successgroup_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/luismfonseca/go-util/sync/successgroup"
	. "github.com/smartystreets/goconvey/convey"
)

func BenchmarkSuccessGroupBestCaseScenario(b *testing.B) {
	for i := 0; i < b.N; i++ {
		group := successgroup.New()
		group.Go(func() error {
			return nil
		})
		group.Go(func() error {
			return nil
		})
		group.Wait()
	}
}

func TestSuccessGroup(t *testing.T) {
	Convey("A successgroup", t, func() {
		group := successgroup.New()

		Convey("should return after it has run the Go function", func() {
			var success bool
			group.Go(func() error {
				<-time.After(10 * time.Millisecond) // simulate an expensive operation
				success = true
				return nil
			})

			err := group.Wait()
			So(err, ShouldBeNil)
			So(success, ShouldBeTrue)
		})

		Convey("should return the last error", func() {
			err1, err2, err3 := errors.New("1"), errors.New("2"), errors.New("3")

			group.Go(func() error {
				return err1
			})
			group.Go(func() error {
				<-time.After(10 * time.Millisecond)
				return err2
			})
			group.Go(func() error {
				<-time.After(20 * time.Millisecond)
				return err3
			})

			err := group.Wait()
			So(err, ShouldEqual, err3)
		})

		Convey("should return immediately if no Go functions were called", func() {
			err := group.Wait()
			So(err, ShouldBeNil)
		})

		Convey("should allow a base context to be passed", func() {
			group, ctx := successgroup.WithContext(context.Background())

			Convey("and it should cancel it if one function was successful", func() {
				var wasCancelled atomic.Value

				group.Go(func() error {
					return errors.New("error")
				})
				group.Go(func() error {
					<-time.After(10 * time.Millisecond)
					return nil
				})
				group.Go(func() error {
					for {
						select {
						case <-ctx.Done():
							wasCancelled.Store(true)
							return nil
						case <-time.After(20 * time.Millisecond):
							return nil
						}
					}
				})

				err := group.Wait()
				So(err, ShouldBeNil)

				// make sure the cancelled part is executed
				<-time.After(10 * time.Millisecond)

				So(wasCancelled.Load(), ShouldBeTrue)
			})

			Convey("and it should cancel even if all have finished", func() {
				var wasCancelled atomic.Value

				group.Go(func() error {
					return errors.New("error")
				})
				group.Go(func() error {
					return errors.New("another error")
				})
				group.Wait()

				select {
				case <-time.After(10 * time.Millisecond):
					wasCancelled.Store(false)
				case <-ctx.Done():
					wasCancelled.Store(true)
				}

				So(wasCancelled.Load(), ShouldBeTrue)
			})
		})
	})
}
