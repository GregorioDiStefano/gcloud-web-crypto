import React, { Component } from 'react';
import ReactDOM from 'react-dom';
import Folder from './components/folders'
import Upload from './components/upload'
import request from 'superagent'
import { Router, Route, Link, browserHistory } from 'react-router'

class App extends Component {
  constructor(props) {
    super(props)
      this.state = { fs: {}, file: {}}
    }

    render() {
      return <h1>Test</h1>
    }
}

ReactDOM.render((
        <Router history={browserHistory}>
            <Route path="/" component={App}/>
              <Route path="upload" component={Upload}/>
              <Route path="dir" component={Folder}/>
        </Router>
      ), document.querySelector('.container'));
