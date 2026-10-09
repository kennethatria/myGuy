'use strict';

const { NodeSDK } = require('@opentelemetry/sdk-node');
const { HttpInstrumentation } = require('@opentelemetry/instrumentation-http');
const { ExpressInstrumentation } = require('@opentelemetry/instrumentation-express');

// With no traceExporter given, the SDK sends traces over OTLP/HTTP to
// OTEL_EXPORTER_OTLP_ENDPOINT (Jaeger), or http://localhost:4318 when unset;
// OTEL_TRACES_EXPORTER=none turns tracing off.
const sdk = new NodeSDK({
  serviceName: 'myguy-chat-service',
  instrumentations: [
    // Health checks aren't worth a trace and would crowd out real requests
    // in Jaeger's in-memory limit.
    new HttpInstrumentation({
      ignoreIncomingRequestHook: (req) => (req.url || '').split('?')[0] === '/health',
    }),
    new ExpressInstrumentation(),
  ],
});

sdk.start();

process.on('SIGTERM', () => {
  sdk.shutdown().finally(() => process.exit(0));
});
