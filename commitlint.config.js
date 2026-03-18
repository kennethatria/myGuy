module.exports = {
    extends: ['@commitlint/config-conventional'],
    rules: {
      'body-max-line-length': [1, 'always', 200],
      'header-max-length': [1, 'always', 150],
    },
  };