- [x] add user id automatically into context for LLM

  - [x] include relevant information like household id

- [x] reply in discord in between agent func calls

  - [x] find a way for the agent func to reply to the bot (caller) and come back (like python yield)
  - [x] reply only with text from LLM (no tools)

- [x] MAKE IT STATELESS!!!

  - [ ] ~~Include all details from LLM in discord message, like tool cals to preserve the history~~
  - [ ] ~~Get last N messages from discord and pass into LLM~~
  - [x] Write llm messages and sessions in db table
  - [x] Read from db instead of messages in memory
  - [x] Keep track of parent messages (for tool calls and results only for now)

- [ ] store transactions first then ask details

  - [x] return created transaction ids from db
  - [x] refactor mapstructure logic with hooks and WeaklyTyped parameter
  - [ ] add update transactions tool
  - [ ] add instructions to system prompt

- [x] Add user and message in single message

- [ ] if message comes from server -> household by default, if comes from pm -> personal by default if not otherwise specified

- [ ] db structure review

- [ ] implement paid/owed functionality

  - [ ] Add other household members to message context
  - [ ] make users only be in a single household
  - [ ] add user default owed amount into household_user
  - [ ] add discord server id into households table to link server to household

- [ ] Add agent loop

  - [ ] Make agent run in a loop instead of just 2 calls, and let it decide when its done
  - [ ] Add a way to cancel execution

- [ ] improve discord interactions

  - [ ] TODO for errors as spoilers
  - [ ] Add tool calls with args and results in a non-intrusive Wy
  - [ ] Handle message mentions correctly
  - [ ] Consider only triggering bot if mentioned

- [ ] tools to get transactions using natural language
- [ ] Add more context from the incoming message, like message time
- [ ] Keep track of token usage per db message
- [ ] improve system prompt

- [ ] improve context management

  - [ ] reset or compact session automatically when a context threshold is reached

- [ ] Access to tools per user

MAYBE

- [ ] Handle simultaneous messages
