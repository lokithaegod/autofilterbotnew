package fsub

import "sync/atomic"

// Stats counters contain no user identifiers and reset on process restart.
// They are not persistent user analytics.
var (
 metricsChecks atomic.Uint64
 metricsPassed atomic.Uint64
 metricsBlocked atomic.Uint64
 metricsErrors atomic.Uint64
 metricsMembershipChecks atomic.Uint64
 metricsMembershipErrors atomic.Uint64
 metricsJoinPrompts atomic.Uint64
 metricsRetries atomic.Uint64
 metricsJoinRequests atomic.Uint64
)

type Stats struct{
 Checks,Passed,Blocked,Errors uint64
 MembershipChecks,MembershipErrors uint64
 JoinPrompts,Retries,JoinRequests uint64
}
func Snapshot() Stats {
 return Stats{
  Checks:metricsChecks.Load(),Passed:metricsPassed.Load(),
  Blocked:metricsBlocked.Load(),Errors:metricsErrors.Load(),
  MembershipChecks:metricsMembershipChecks.Load(),
  MembershipErrors:metricsMembershipErrors.Load(),
  JoinPrompts:metricsJoinPrompts.Load(),Retries:metricsRetries.Load(),
  JoinRequests:metricsJoinRequests.Load(),
 }
}
func RecordRetry(){metricsRetries.Add(1)}
func RecordJoinRequest(){metricsJoinRequests.Add(1)}
