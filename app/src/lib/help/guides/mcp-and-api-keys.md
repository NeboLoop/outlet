# Connect an AI assistant over MCP, and API keys

Outlet can be run by an AI assistant such as Claude or ChatGPT, through MCP (Model Context Protocol, a standard way for AI assistants to use other software). The assistant can create lists, write campaigns, find prospects, draft outreach and explain problems, all by asking. This guide also explains Outlet's two kinds of keys.

## Two kinds of keys

| Key | What it's for | Where |
| --- | --- | --- |
| **Brand API key** | Your apps sending email and subscribing people | The brand's **Settings**, **General**, **API Key** |
| **MCP key** | AI assistants managing Outlet | Outlet's **Settings**, **API**, **MCP keys** |

They don't mix: an MCP key can't send email, and a brand API key can't manage Outlet.

## Connect an AI assistant (the easy way)

1. Open Outlet's own **Settings**: on the **Brands** screen, click your initial at the top right and choose **Settings**. The **MCP** tab opens.
2. Copy **Your MCP Server URL**. It ends in `/mcp`.
3. Under **Setup Instructions**, pick your assistant: **Claude Desktop**, **ChatGPT**, **Claude Code** or **Cursor / IDE**, and follow its steps.
4. When the assistant connects, it opens a browser window. Sign in to Outlet there and allow access.

No key is needed this way: the assistant signs in as you (this is called OAuth), and can do what your account can do.

**Available Tools** on the same page lists everything the assistant can do, by group.

## Connect with an MCP key

Some tools can't open a sign-in window. Give them an MCP key instead.

1. In Outlet's **Settings**, open the **API** tab.
2. Under **MCP keys**, press **Create Key**, type a **Key Name** such as "Claude at work", and press **Create Key**.
3. Copy the key now. Outlet shows it once: "This is the only time you'll see this key."
4. In the tool, use it as a bearer token: the header `Authorization: Bearer` followed by the key.

To stop a key working, press the red button on its row and then **Revoke**. Revoking can't be undone.

## What an assistant can't do

On purpose, an assistant can't turn on outreach. Only the brand's admin can, in the app. An assistant also can't spend money at a mailbox service without your yes: it must show you the quote first.

## The brand API key

Your apps use the brand's API key to send transactional email and subscribe people. See [Transactional email from your app](/help/transactional-email).

1. Open the brand's **Settings**, **General**.
2. Under **API Key**, press the copy button.
3. **Regenerate Key** makes a new one. The old key stops working at once.

## Common problems

- **The assistant says it isn't allowed**: your account lacks the brand role for that action. See [Users and roles](/help/users-and-roles).
- **"Invalid API key" from your app**: you used an MCP key, or the brand key was regenerated.
- **The sign-in window doesn't open**: use an MCP key instead.
