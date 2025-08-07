- [x] add user id automatically into context for LLM
  - [x] include relevant information like household id

- [ ] reply in discord in between agent func calls
  - [ ] find a way for the agent func to reply to the bot (caller) and come back (like python yield)
  - [ ] reply only with text from LLM (no tools)

- [] store transactions first then ask details
  - return created transaction ids from db
  - add update transactions tool
  - add instructions to system prompt

- [ ] MAKE IT STATELESS!!!
  - [ ] Get last N messages from discord and pass into LLM
  - [ ] overall improve context management

- [ ] if message comes from server -> household by default, if comes from pm -> personal by default if not otherwise specified

- [ ] implementpaid/owed functionality
  - [ ] Add other household members to message context
  - [ ] make users only be in a single household
  - [ ] add user default owed amount into household_user
  - [ ] add discord server id into households table to link server to household

- [ ] improve discord interactions
  - [ ] TODO for errors as spoilers
  - [ ] Add tool calls with args and results in a non-intrusive Wy

- [] tools to get transactions using natural language
- [ ] Add more context from the incoming message, like message time
- [ ] improve system prompt

MAYBE
- [ ] Handle simultaneous messages 