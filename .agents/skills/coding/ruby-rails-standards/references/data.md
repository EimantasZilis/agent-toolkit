# Rails data and persistence

## Apply this first

Treat the database as a concurrency boundary and the schema as part of the
application contract. Use model validations for useful feedback, but enforce
invariants that must hold under concurrent writes with database constraints.

- Add indexes and unique, foreign-key, nullability, and check constraints for
  invariants the database must always enforce.
- Make migrations forward-only, reversible where practical, and safe for the
  deployed data volume; separate backfills from locking schema changes when
  needed.
- Use transactions for an atomic unit of related writes and keep external I/O
  out of the transaction body.
- Query only the columns and rows needed; inspect query plans and avoid N+1
  loads with deliberate eager loading.
- Use `Time.current`/`Time.zone` for application time and make timezone
  assumptions explicit.
- Treat callbacks as lifecycle hooks, not a general workflow engine; prefer
  `after_commit` for work that depends on durable data.

```ruby
# DO: pair application feedback with a database invariant.
class Membership < ApplicationRecord
  validates :user_id, uniqueness: { scope: :team_id }
end

# migration
add_index :memberships, [:team_id, :user_id], unique: true
```

```ruby
# DO NOT: assume validation alone prevents a concurrent duplicate.
validates :email, uniqueness: true
# with no matching unique database index
```

Allowed exceptions: a deliberately denormalized read model, legacy table, or
database that cannot support a constraint may require application-only checks;
document the race and compensating control. A migration may use raw SQL when
the adapter API cannot express a required operation, with adapter/version
coverage in tests.

## Quick checklist

- [ ] Schema invariants are enforced at the database boundary where possible.
- [ ] Migration locking, backfill size, rollback, and deploy order are safe.
- [ ] Transaction scope is minimal and external side effects occur after commit.
- [ ] Queries avoid N+1 behavior and select appropriate data.
- [ ] Concurrent and invalid-data paths are tested.

## Sources and decisions

- [Active Record Basics](https://guides.rubyonrails.org/active_record_basics.html), accessed 2026-09-30.
- [Active Record Validations](https://guides.rubyonrails.org/active_record_validations.html), accessed 2026-09-30.
- [Active Record Callbacks](https://guides.rubyonrails.org/active_record_callbacks.html), accessed 2026-09-30.
- Rails documents that validations can be bypassed by some persistence methods;
  the database-constraint rule is the pack's explicit concurrency safeguard.

