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
            const notebook = Jupyter.notebook.toJSON();
            if ('cells' in notebook && notebook.cells.length > 0) {
              // Drop the cell outputs.
              notebook.cells = notebook.cells.map(function(cell) {
                cell.outputs = [];
                return cell;
              });
            }
            const content = JSON.stringify(notebook, null, 2);
            const url = configuration.upload_it_server_url;
            const formdata = new FormData();
            const blob = new Blob([content], { type: "application/x-ipynb+json"});
            formdata.set("notebook", blob);
            window.console.log("Uploading ", Jupyter.notebook.notebook_path, " to ", url, formdata);
            $.ajax({
              url: url,
              xhrFields: {withCredentials: true},
              data: formdata,
              contentType: false,
              processData: false,
              method: "POST",
              success: function(data, status, jqXHR) {
                // Open the report in a new tab.
                let reportURL = new URL(url);
                // Expect the report location to provided in a custom HTTP header.
                let reportLocation = jqXHR.getResponseHeader("X-Report-Url");
                if (reportLocation != null) {
                  reportURL.pathname = reportLocation;
                } else {
                  // If header was not provided, try to parse the document
                  // and extract the first link.
                  window.console.log("did not receive X-Report-Url, received data ", data);
                  const parser = new DOMParser();
                  const htmlDoc = parser.parseFromString(data, 'text/html');
                  const links = htmlDoc.getElementsByTagName('a');
                  if (links.length == 0) {
                    windows.console.error("did not find any links in the response");
                    return;
                  }
                  reportURL = links[0].href;
                }
                window.console.log("Upload OK, opening report at ", reportURL.toString());
                window.open(reportURL, '_blank');
              },
              error: function(jqXHR, status, err) {
                if (jqXHR.status == 401) {
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
