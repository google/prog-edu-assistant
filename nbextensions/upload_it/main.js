/**
 * @license
 * Copyright 2019 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

define([
  'jquery',
  'base/js/namespace',
  'base/js/dialog',
], function($, Jupyter, dialog) {
  "use strict";

  // Default values for the configuration parameters.
  const configuration = {
    "upload_it_server_url": "http://localhost:8000/upload"
  };

  function initialize() {
    updateConfig();
    Jupyter.toolbar.add_buttons_group([
      Jupyter.keyboard_manager.actions.register({
        "help": "Upload notebook",
        "icon": "fa-check-square",
        "handler": showUploadDialog
      }, "upload-notebook", "upload_it")
    ]);
  }

  function updateConfig() {
    // If the notebook has configuration overrides, pick them up.
    const config = Jupyter.notebook.config;
    for (const key in configuration) {
      if (config.data.hasOwnProperty(key)) {
        configuration[key] = config.data[key];
      }
    }
  }

  // Returns a JQuery object with the dialog.
  function buildUploadDialog() {
    let $uploadDialog = $("#upload_it_dialog");
    if ($uploadDialog.length > 0) {
      return $uploadDialog;
    }
    $uploadDialog = $("<div>")
      .attr("id", "upload_it_dialog");
    return $uploadDialog;
  }

  function showUploadDialog() {
    const modal = dialog.modal({
      show: false,
      title: "Upload notebook",
      notebook: Jupyter.notebook,
      keyboard_manager: Jupyter.notebook.keyboard_manager,
      body: buildUploadDialog(),
      buttons: {
        "Upload": {
          "class": "btn-primary",
          "click": function () {
            const notebook = Jupyter.notebook;
            const url = configuration.upload_it_server_url;
            const formdata = new FormData();
            const content = JSON.stringify(Jupyter.notebook.toJSON(), null, 2);
            const blob = new Blob([content], { type: "application/x-ipynb+json"});
            formdata.set("notebook", blob);
            window.console.log("Uploading ", notebook.notebook_path, " to ", url, formdata);
            $.ajax({
              url: url,
              xhrFields: {withCredentials: true},
              data: formdata,
              contentType: false,
              processData: false,
              method: "POST",
              success: function(data, status, jqXHR) {
                // Open the report in a new tab.
                const reportURL = new URL(url);
                reportURL.pathname = data;
                window.console.log("Upload OK, opening report at ", reportURL.toString());
                window.open(reportURL, '_blank');
              },
              error: function(jqXHR, status, err) {
                if (err == "Unauthorized") {
                  window.console.log("Unauthorized, attempting login");
                  const loginURL = new URL(configuration.upload_it_server_url);
                  loginURL.pathname = '/login';
                  window.open(loginURL, '_blank');
                  return;
                }
                window.console.log("Upload failed", status, err);
              }
            });
          }
        },
        "done": {}
      }
    });
    modal.attr('id', 'upload_it_modal');
    modal.modal('show');
  }

  function loadJupyterExtension() {
    return Jupyter.notebook.config.loaded.then(initialize);
  }

  return {
    load_ipython_extension: loadJupyterExtension,
    load_jupyter_extension: loadJupyterExtension
  };

});  // End of module definition.
