# Rails application structure

## Apply this first

Keep framework boundaries legible. Controllers translate HTTP into application
operations, models protect persistence invariants, jobs own asynchronous work,
and domain/service objects own multi-step business workflows when the behavior
does not fit one framework object.

- Follow Rails naming and file/constant alignment so Zeitwerk can autoload and
  reload the code.
- Keep controllers thin: authenticate, authorize, parse permitted parameters,
  call an operation, and render or redirect.
- Prefer explicit query objects, scopes, or service objects over callbacks and
  model methods that conceal external side effects.
- Keep views presentation-focused; do not put database writes or authorization
  decisions in templates.
- Use `ApplicationJob` and pass stable identifiers or serializable arguments;
  re-load current records inside the job.

```ruby
# DO: controller coordinates the boundary.
def create
  article = CreateArticle.call(actor: current_user, attributes: article_params)
  redirect_to article_path(article), status: :see_other
end

private

def article_params
  params.expect(article: [:title, :body])
end
```

```ruby
# DO NOT: put authorization, writes, and unrelated notification side effects
# in a view or a broad before_action used by every action.
```

Allowed exceptions: a small CRUD controller may use direct Active Record calls;
a callback may be appropriate for an intrinsic local invariant. For external
side effects, use an explicit operation or transaction callback and test the
ordering.

## Quick checklist

- [ ] File paths, constants, and namespaces agree with the loader.
- [ ] HTTP, domain, persistence, and presentation responsibilities are clear.
- [ ] Parameters are explicitly permitted and authorization is visible.
- [ ] Jobs are idempotent or document their delivery/retry behavior.
- [ ] Changed behavior has a focused test.

## Sources and decisions

- [Rails autoloading and reloading](https://guides.rubyonrails.org/autoloading_and_reloading_constants.html), accessed 2026-09-30.
- [Active Job Basics](https://guides.rubyonrails.org/active_job_basics.html), accessed 2026-09-30.
- [RuboCop Rails](https://docs.rubocop.org/rubocop-rails/latest/index.html), accessed 2026-09-30.
- Rails' loader and job contracts are official framework behavior; the
  controller/service split is a maintainability convention and yields to a
  repository's established architecture.

