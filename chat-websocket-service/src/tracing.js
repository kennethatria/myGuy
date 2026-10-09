'use strict';

const { NodeSDK } = require('@opentelemetry/sdk-node');
const { HttpInstrumentation } = require('@opentelemetry/instrumentation-http');
const { ExpressInstrumentation } = require('@opentelemetry/instrumentation-express');

// With no traceExporter given, the SDK sends traces over OTLP/HTTP to
// OTEL_EXPORTER_OTLP_ENDPOINT (Tempo), or http://localhost:4318 when unset;
// OTEL_TRACES_EXPORTER=none (production) turns tracing off.
const sdk = new NodeSDK({
  serviceName: 'myguy-chat-service',
  instrumentations: [
    new HttpInstrumentation(),
    new ExpressInstrumentation(),
  ],
});

sdk.start();

process.on('SIGTERM', () => {
  sdk.shutdown().finally(() => process.exit(0));
});
