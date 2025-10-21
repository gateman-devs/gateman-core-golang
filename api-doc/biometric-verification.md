---
title: 'Biometric Verification'
description: 'Verify identities using face recognition and liveness detection'
---

Verify user identities securely with advanced face recognition. Confirm if two faces match or detect liveness to prevent spoofing.

## Endpoints

### Face Comparison
Compare two face images to verify if they belong to the same person.

- **Endpoint:** `POST /api/public/v1/biometric/face-comparison`
- **Use Cases:** Login verification, account security, identity checks

### Liveness Check
Detect if a face image is from a real person or a fake (photo, video, or mask).

- **Endpoint:** `POST /api/public/v1/biometric/liveness-check`
- **Use Cases:** Prevent fake accounts, secure transactions, age verification

## Quick Start

Example: Compare two faces

```javascript
const response = await fetch('https://api.gateman.io/api/public/v1/biometric/face-comparison', {
  method: 'POST',
  headers: {
    'x-api-key': 'your-api-key',
    'x-app-id': 'your-app-id',
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    image1: 'https://example.com/face1.jpg',
    image2: 'https://example.com/face2.jpg',
    threshold: 0.8
  })
});

const result = await response.json();
console.log(result.body.is_match); // true or false
```

## Features

- High accuracy with AI models
- Fast processing
- Supports URLs and base64 images
- Enterprise security
- Scalable for high volume

## Best Practices

- Use HTTPS
- Ensure good image quality
- Handle errors properly
- Set appropriate thresholds
- Monitor usage

