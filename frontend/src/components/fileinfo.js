import React, { Component } from 'react'
import request from 'superagent'
import { BootstrapTable, TableHeaderColumn } from 'react-bootstrap-table';


class FileInfo extends Component {
  constructor(props) {
    super(props)
    this.setState({ filedata: this.props.fi})
    console.log("fileinfo init")
}

removeFile = (e) => {
  var uuid = this.props.fileid
  request
  .del('http://127.0.0.1:3000/file/' + uuid)
  .end(function(err, res){
    console.log(err, res)
  });
}

render() {
  console.log("rendering fileinfo")
  var fi = "tst"
  console.log("here: ", this.props.fileid)
  if (this.state && typeof this.state.fs !== 'undefined') {
    fi = this.state.fs
  }

  return (
    <div id="fileInfo">
    <form>
      <fieldset>
        <legend>File detail</legend>
          <table>
            <tr>
              <th>Filename</th>
              <td>{this.props.fileName}</td>
            </tr>
            { this.props.fileDescription ?
              <tr>
                <th>Description</th>
                <td>{this.props.fileDescription}</td>
              </tr> : null }
            <tr>
              <th>Type</th>
              <td>{this.props.fileType}</td>
            </tr>
            <tr>
              <th>Size</th>
              <td>{this.props.fileSize}</td>
            </tr>
            <tr>
              <th>MD5</th>
              <td>{this.props.fileMD5Hash}</td>
            </tr>
            <tr>
              <th>Uploaded</th>
              <td>{this.props.fileUploadDate}</td>
            </tr>
          </table>
      </fieldset>
    </form>
    </div>
  );
}
}

export default FileInfo
