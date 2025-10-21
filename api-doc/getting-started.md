---
title: 'Getting Started'
description: 'Quick guide to start using Gateman API'
---

To use the Gateman API, you'll need a Gateman account. If you don't have one yet, you can create an account [here](https://gateman.io).

Once your account is set up, you can access the API using the following base URLs:

- **Production Environment**: `https://api.gateman.io`  
- **Sandbox Environment**: `https://sandbox.gateman.io`  

The Sandbox environment is recommended for testing and development, while the Production environment is for live applications.

## Authentication

All Gateman API endpoints require authentication using an **API Key** and **API ID**. These credentials must be included in the headers of every request.

You can retrieve your API Key and API ID from the **Settings** page on your [Gateman Dashboard](https://gateman.io).

### Example Request Headers

```json
{
  "x-api-key": "your-sandbox-api-key",
  "x-app-id": "your-sandbox-app-id"
}
```

