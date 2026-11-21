package core

import(
 "context"
 "errors"
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"
 "time"
)

func TestHealthChecks(t *testing.T){
 good:=checkHealth(context.Background(),true,func(context.Context)error{return nil},func()error{return nil})
 if !good.Healthy(){t.Fatal("expected all checks passing")}
 down:=checkHealth(context.Background(),true,func(context.Context)error{return errors.New("db down")},func()error{return nil})
 if down.Healthy()||down.Mongo{t.Fatal("db failure must fail health")}
 missing:=checkHealth(context.Background(),false,func(context.Context)error{return nil},func()error{return nil})
 if missing.Healthy(){t.Fatal("polling must be started")}
 ctx,cancel:=context.WithTimeout(context.Background(),10*time.Millisecond);defer cancel()
 late:=checkHealth(ctx,true,func(context.Context)error{<-ctx.Done();return ctx.Err()},func()error{<-ctx.Done();return ctx.Err()})
 if late.Healthy(){t.Fatal("timeout must fail")}
}
func TestHTTPHealthUnready(t *testing.T){
 // No running bot during unit tests: report 503, never return false UP.
 r:=httptest.NewRequest(http.MethodGet,"/healthz",nil)
 w:=httptest.NewRecorder()
 HealthHTTP(w,r)
 if w.Code!=http.StatusServiceUnavailable {t.Fatalf("expected 503, got %d",w.Code)}
 if !strings.Contains(w.Body.String(),"not healthy"){t.Fatalf("bad response %q",w.Body.String())}
 head:=httptest.NewRequest(http.MethodHead,"/",nil)
 w=httptest.NewRecorder();HealthHTTP(w,head)
 if w.Code!=503{t.Fatalf("HEAD must check same health: %d",w.Code)}
}
