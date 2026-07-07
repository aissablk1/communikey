#!/usr/bin/env node
// index.js — serveur MCP communikey-browsers (stdio). Câble TOOLS + handleTool
// (tools.js) au SDK MCP. Toute erreur d'un outil est renvoyée en isError (jamais
// une exception non gérée qui tue le serveur).

import { Server } from "@modelcontextprotocol/sdk/server/index.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { ListToolsRequestSchema, CallToolRequestSchema } from "@modelcontextprotocol/sdk/types.js";
import { TOOLS, handleTool } from "./tools.js";

const server = new Server(
  { name: "communikey-browsers", version: "0.1.0" },
  { capabilities: { tools: {} } }
);

server.setRequestHandler(ListToolsRequestSchema, async () => ({ tools: TOOLS }));

server.setRequestHandler(CallToolRequestSchema, async (req) => {
  try {
    return await handleTool(req.params.name, req.params.arguments ?? {});
  } catch (e) {
    return { content: [{ type: "text", text: `Erreur: ${e.message}` }], isError: true };
  }
});

const transport = new StdioServerTransport();
await server.connect(transport);
