- [x] add user id automatically into context for LLM

  - [x] include relevant information like household id

- [x] reply in discord in between agent func calls

  - [x] find a way for the agent func to reply to the bot (caller) and come back (like python yield)
  - [x] reply only with text from LLM (no tools)

- [x] MAKE IT STATELESS!!!

  - [x] ~~Include all details from LLM in discord message, like tool cals to preserve the history~~
  - [x] ~~Get last N messages from discord and pass into LLM~~
  - [x] Write llm messages and sessions in db table
  - [x] Read from db instead of messages in memory
  - [x] Keep track of parent messages (for tool calls and results only for now)

- [x] db structure review

- [x] hash per transaction

  - [x] handle tool calls errors correctly (see TODO)
  - [x] handle sql duplicate error gracefully

- [x] improve transaction create + update logic

  - [x] return created transaction ids from db
  - [x] refactor mapstructure logic with hooks and WeaklyTyped parameter
  - [x] add update transactions tool (or rewrite create to be upsert)
  - [x] add instructions to system prompt

- [x] Add user and message in single message
- [x] Add more context from the incoming message, like message time

- [x] if message comes from server -> household by default, if comes from pm -> personal by default if not otherwise specified
  - [x] Link household to discord channel ID
  - [ ] fix private messages
- [ ] tools to get transactions using natural language
- [ ] improve system prompt
- [ ] csv report of transactions by date range
- [ ] INVESTIGATE: extract timestamp from picture metadata

- [ ] improve discord interactions

  - [ ] Handle private messages correctly
  - [ ] TODO for errors as spoilers
  - [ ] Add tool calls with args and results in a non-intrusive way
  - [ ] Handle message mentions correctly
  - [ ] Consider only triggering bot if mentioned

- [ ] **RELEASE!!!!**

  - [ ] Deploy to server
  - [ ] Set up CI/CD
  - [ ] maybe use docker swarm / k8s

- [ ] Improve logging and monitoring

  - [ ] Use structured logging https://go.dev/blog/slog
  - [ ] Set up Grafana + Prometheus + Loki

- [ ] Keep track of token usage per db message
- [ ] add session summary (like what chat apps do)

- [ ] improve context management

  - [ ] reset or compact session automatically when a context threshold is reached

- [ ] Budget tracking

  - [ ] DB to store budget allocation per user (or percentage split) (maybe automatic based on income?) (or maybe assign categories to people)
  - [ ] Add some default budget categories
  - [ ] Allocate budget using natural language
  - [ ] Automatically get category from similar transactions

- [ ] implement paid/owed functionality

  - [ ] Add other household members to message context
  - [ ] make users only be in a single household
  - [ ] add user default owed amount into household_user
  - [ ] add discord server id into households table to link server to household

- [ ] command to "sign up"
  - [ ] Register household name and guild id
  - [ ] Register members of the server as users and assign to household

MAYBE

- [ ] Handle simultaneous messages
- [ ] Add agent loop
  - [ ] Make agent run in a loop instead of just 2 calls, and let it decide when its done
  - [ ] Add a way to cancel execution
- [ ] Access to tools per user
