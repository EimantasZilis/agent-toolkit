# Rails security

## Apply this first

Treat every request, header, cookie, uploaded file, webhook, and job argument
as untrusted until validated and authorized. Make the security boundary visible
in the code and test it.

- Permit only the parameters needed by the action; do not pass raw `params` to
  models or service objects.
- Authenticate and authorize the actor for every protected operation; do not
  rely on obscurity, UI hiding, or an object identifier alone.
- Use parameterized Active Record queries and safe view escaping; do not build
  SQL or HTML with untrusted interpolation.
- Keep credentials in the configured secret store/environment, rotate leaked
  values, and redact tokens, cookies, passwords, and personal data from logs.
- Preserve CSRF and same-origin protections unless the endpoint is a deliberate
  non-browser boundary with an equivalent authentication design.
- Validate uploads, redirects, serialization, and webhook signatures at their
  boundary.

```ruby
# DO: authorize the loaded record and permit an explicit shape.
def update
  article = current_user.articles.find(params[:id])
  article.update!(article_params)
end

def article_params
  params.expect(article: [:title, :body])
end
```

```ruby
# DO NOT: interpolate request data into SQL or trust an ID without ownership.
Article.where("title = '#{params[:title]}'")
Article.find(params[:id]).update!(params[:article])
```

Allowed exceptions: raw SQL is acceptable for a reviewed, parameterized query
that the relation API cannot express; an API endpoint may disable browser CSRF
handling only when it uses an explicit non-cookie authentication and has tests
for replay, authorization, and origin assumptions.

## Quick checklist

- [ ] Authentication, authorization, and tenant/resource scope are enforced.
- [ ] Parameters, uploads, redirects, and webhook inputs are constrained.
- [ ] SQL, HTML, shell, and serialization boundaries are safe.
- [ ] Secrets and personal data are absent from code, logs, fixtures, and tests.
- [ ] Security-sensitive behavior has a regression test.

## Sources and decisions

- [Rails Security Guide](https://guides.rubyonrails.org/security.html), accessed 2026-09-30.
- [Rails controller testing and request boundaries](https://guides.rubyonrails.org/testing.html), accessed 2026-09-30.
- The Rails Security Guide is the primary framework source; the concrete review
  checklist generalizes its injection, session, CSRF, and authorization risks
  to all request and background-job boundaries.

