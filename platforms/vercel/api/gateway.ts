/// <reference types="@vercel/node" />

import type { VercelRequest, VercelResponse } from '@vercel/node';

// Placeholder for WASM module loading
// In production, the Go WASM binary will be loaded from the filesystem

export default async function handler(req: VercelRequest, res: VercelResponse) {
  const path = req.url || '/';

  // CORS headers
  res.setHeader('Access-Control-Allow-Origin', '*');
  res.setHeader('Access-Control-Allow-Methods', 'GET, POST, PUT, PATCH, DELETE, OPTIONS');
  res.setHeader('Access-Control-Allow-Headers', 'Content-Type, Authorization, X-Bendy-Timestamp');

  if (req.method === 'OPTIONS') {
    return res.status(204).end();
  }

  // Health check endpoint
  if (path === '/health' || path === '/api/v1/health') {
    return res.status(200).json({
      status: 'ok',
      version: process.env.VERSION || '0.1.0',
      service: 'bendy-file-gateway',
    });
  }

  // Serve admin dashboard for non-API routes
  if (!path.startsWith('/api/') && !path.startsWith('/admin/api/')) {
    return res.status(200).send('Bendy File Gateway - Admin dashboard will be served here');
  }

  // Placeholder: Gateway is initializing
  return res.status(501).json({
    error: 'not_implemented',
    message: 'Gateway is initializing',
  });
}
