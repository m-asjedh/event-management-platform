import express from "express";
import { toNodeHandler } from "better-auth/node";

import { auth } from "./auth.js";

const port = Number(process.env.AUTH_PORT ?? 3001);

const app = express();

// Route pattern is Express 5, which needs a named wildcard: the Express 4 form
// "/api/auth/*" throws at startup under path-to-regexp v8.
//
// Registered before express.json(), because the handler needs the raw body stream.
app.all("/api/auth/*splat", toNodeHandler(auth));

app.use(express.json());

// Used by the compose healthcheck, so nothing depending on this service starts before
// it can serve a request.
app.get("/healthz", (_req, res) => {
  res.json({ status: "ok" });
});

app.listen(port, () => {
  console.log(`auth listening on ${port}`);
});
