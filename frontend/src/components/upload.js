import React from 'react'
import {Fieldset, Field, createValue} from 'react-forms'
import Progress from 'react-progressbar'
import superagent from 'superagent'
import SweetAlert from 'sweetalert-react';
import Header from './header.js';
import Search from './search.js';

class Upload extends React.Component {

  constructor(props) {
    super(props)
    let formValue = createValue({
      value: props.value
    })
    this.state = {formValue, uploadProgress: 0.0}
    this.tags = {}
  }

  componentDidMount() {
    this.refs.folderUpload.directory = true;
    this.refs.folderUpload.webkitdirectory = true;
  }

  onUpload = (e) => {
    e.preventDefault();
    this.setState({"uploadError": false, "uploadComplete": false})

    var form = document.getElementById("uploadForm"),
        fd  = new FormData(form),
        self = this;

    if (this.tags.length > 0) {
      let tagText = this.tags.map(function(tag) {return tag.text;});
      fd.append("tags", tagText.join(","))
    }

    superagent.post('/auth/file/')
              .on('progress', function(e) {
                  self.setState({"uploadProgress": e.percent})
               })
               .send(fd)
               .end(function(err, response) {
                    self.setState({"uploadProgress": 0})
                  if (err) {
                    console.log("response.text: ", response.text)
                    self.setState({"uploadError": true, "uploadErrorMessage": response.text})
                  } else {
                    self.setState({"uploadComplete": true})
                  }
               });
    }

  tagsCallback = (tags) => {
    this.tags = tags
  }

  render() {
    return (
    <div>
      <Header/>
      <form id="uploadForm" className="form-horizontal" action="/auth/file/" method="post" encType="multipart/form-data" ref="uploadForm">
        <Fieldset formValue={this.state.formValue}>

          <div className="form-group">
            <label className="col-md-4 control-label" htmlFor="textinput">Description</label>
            <div className="col-md-5">
            <input id="textinput" name="description" type="text" placeholder="File description" className="form-control input-md"/>
            <span className="help-block"> </span>
            </div>
          </div>

          <div className="form-group">
            <label className="col-md-4 control-label" htmlFor="filebutton">Files to upload</label>
            <div className="col-md-4">
              <input id="filebutton" name="file" className="input-file" type="file" multiple/>
            </div>
          </div>

          <div className="form-group">
            <label className="col-md-4 control-label" htmlFor="folderupload">Folder to upload</label>
            <div className="col-md-4">
              <input id="folderupload" ref="folderUpload" name="file" className="input-file" type="file" webkitdirectory="" directory="" multiple/>
            </div>
          </div>

          <div className="form-group">
            <label className="col-md-4 control-label" htmlFor="virtfolder">Folder</label>
            <div className="col-md-5">
            <input id="virtfolder" name="virtfolder" type="text" placeholder="Virtual folder" className="form-control input-md"/>
            <span className="help-block"></span>
            </div>
          </div>
          <div className="form-group">
            <label className="col-md-4 control-label" htmlFor="taginput">Tags</label>
            <div className="col-md-4">
                <Search tagsCallback={this.tagsCallback}/>
            </div>
          </div>
          <div className="form-group">
            <label className="col-md-4 control-label" htmlFor="singlebutton"></label>
            <div className="col-md-4">
              <button id="singlebutton" name="singlebutton" className="btn btn-primary" disabled={this.state.uploadProgress > 0} onClick={this.onUpload}>Upload</button>
            </div>
          </div>
        </Fieldset>

      <Progress completed={this.state.uploadProgress} />
      <SweetAlert show={this.state.uploadComplete} title="Upload success" onConfirm={() => { this.setState({ uploadComplete: false}); } }/>
      <SweetAlert show={this.state.uploadError} title="Upload failed" text={this.state.uploadErrorMessage} onConfirm={() => { this.setState({ uploadError: false}); } }/>

      </form>
      </div>
    )
  }
}

export default Upload
