# Human Syntax Quick Reference

## Application

```
app <Name> is a <web|mobile|desktop|api> application
```

## Data Models

```
data <Name>:
  has a <field> which is <type>           # required field
  has an optional <field> which is <type> # nullable
  has a <field> which is unique <type>    # unique constraint
  has a <field> which is encrypted <type> # encrypted at rest
  has a <field> which is either "a" or "b"  # enum
  has a <field> which defaults to <value>   # default
  belongs to a <Data>                     # many-to-one
  has many <Data>                         # one-to-many
  has many <Data> through <JoinData>      # many-to-many
```

**Types:** `text`, `number`, `decimal`, `boolean`, `date`, `datetime`, `email`, `url`, `file`, `image`, `json`

## Pages

```
page <Name>:
  show <what>                             # render content
  show a list of <data>                   # render collection
  show each <item>'s <field> and <field>  # specify fields
  show <data> in a <card|table|grid|list> # specify layout
```

## Events & Interactions

```
clicking <element> does <action>          # click handler
clicking <element> navigates to <page>    # navigation
clicking <element> opens <thing>          # modal/panel
typing in <element> does <action>         # input handler
hovering over <element> shows <thing>     # hover effect
pressing <key> does <action>              # keyboard shortcut
scrolling to bottom loads more <data>     # infinite scroll
dragging <element> reorders the list      # drag and drop
```

## Forms & Inputs

```
there is a text input for <purpose>
there is a search bar that filters <data>
there is a dropdown to select <options>
there is a checkbox for <purpose>
there is a date picker for <purpose>
there is a file upload for <purpose>
there is a form to create <data>
```

## Conditional Display

```
if <condition>, show <thing>
if no <data> match, show "<message>"      # empty state
while loading, show a spinner             # loading state
if there is an error, show the error message  # error state
```

## Components

```
component <Name>:
  accepts <prop> as <type>
  <content statements>
```

## APIs

```
api <Name>:
  requires authentication
  accepts <fields>
  check that <validation>
  fetch <data> from <source>
  create a <Data> with <fields>
  update the <Data>
  delete the <Data>
  if <condition>, respond with "<message>"
  respond with <data>
  sort by <field> newest first
  support filtering by <field>
  paginate with <count> per page
```

## Security

```
authentication:
  method JWT tokens that expire in <duration>
  method <Provider> OAuth
  passwords are hashed with <algorithm>
  rate limit <scope> to <limit> per <period>
  sanitize all text inputs against XSS
  enable CORS only for <domain>
```

## Policies

```
policy <Name>:
  can <permission>
  cannot <restriction>
```

## Database

```
database:
  use <PostgreSQL|MySQL|MongoDB|SQLite>
  index <Data> by <field>
  backup daily at <time>
```

## Workflows

```
when <event>:
  <action sequence>
  after <delay>, <action>
  notify all <audience> of <event>
```

## Theme

```
theme:
  primary color is <color>
  secondary color is <color>
  font is <font> for body and <font> for headings
  border radius is <sharp|smooth|rounded|pill>
  dark mode is supported
  spacing is <compact|comfortable|spacious>
  use <design_system> design system
```

## Build Targets

```
build with:
  frontend using <React|Vue|Angular|Svelte> with TypeScript
  backend using <Node|Python|Go> with <Express|FastAPI|Gin>
  database using <PostgreSQL|MySQL|MongoDB>
  deploy to <Docker|AWS|GCP|Vercel>
```

## Architecture

```
architecture: <monolith|microservices|serverless>

service <Name>:
  handles <responsibilities>
  talks to <Service> to <purpose>
```

## Error Handling

```
if <service> is unreachable:
  retry <count> times with <delay>
  if still failing, respond with "<message>"
  alert the <team> via <channel>
```

---

Run `human explain <topic>` for detailed reference on any section.
Run `human syntax --search "<term>"` to find specific patterns.
