module.exports = function(grunt) {
  var extend = require('util')._extend;
  require('load-grunt-tasks')(grunt);

  // Project configuration.
  grunt.initConfig({
    pkg: grunt.file.readJSON('package.json'),
    uglify: {
      options: {
        banner: '/** <%= grunt.template.today("yyyy-mm-dd") %> */\n',
        mangle: true,
        preserveComments: false,
      },
      build: {
        files: {
          'scripts/inject.min.js': ['src/inject.js'],
          'scripts/start_event.min.js': ['src/start_event.js'],
          'scripts/finish_event.min.js': ['src/finish_event.js'],
          'scripts/play_stream_event.min.js': ['src/play_stream_event.js'],
          'scripts/pause_stream_event.min.js': ['src/pause_stream_event.js']
        }
      }
    },
    shell: {
      deploy: {
        command: "goapp deploy"
      }
    },
    gae_config: {
      dev: {
        application: "glitchapi-dev"
      },
      prod: {
        application: "glitchapi"
      }
    }
  });

  grunt.registerMultiTask('gae_config', 'Creates GAE config for deployment environment', function() {
    var options = this.options({
      configFile: 'gae_config.json',
    });
    var config = grunt.file.readJSON(options.configFile);
    var yaml = grunt.file.read('app_template.yaml');
    var output = grunt.template.process(yaml, { data: extend(config[this.target], { application: this.data.application }) });
    grunt.file.write("app.yaml", output);
    grunt.log.writeln("Created app.yaml for deployment");
  });

  // Default task(s).
  grunt.registerTask('default', ['uglify:build']);

  grunt.registerTask('deploy:dev', ['uglify:build', 'gae_config:dev', 'shell:deploy']);

  grunt.registerTask('deploy:prod', ['uglify:build', 'gae_config:prod', 'shell:deploy']);

};