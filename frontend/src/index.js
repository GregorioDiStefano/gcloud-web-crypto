import React, { Component } from 'react';
import ReactDOM from 'react-dom';
import Folder from './components/folders'
import Upload from './components/upload'
import request from 'superagent'
import Login from './components/login'
import { Router, Route, Redirect, Link, browserHistory } from 'react-router'

class App extends Component {
  constructor(props) {
    super(props)
  }

    userIsAuthenticated = (nextState, replace) => {
        if (localStorage.getItem("jwt") == null) {
          replace(`/login`)
        }
    }

    render() {
      return (
      <Router history={browserHistory}>
            <Route path="/" component={Folder} onEnter={this.userIsAuthenticated}/>
            <Router path="/login" component={Login} />
            <Route path="upload" component={Upload} onEnter={this.userIsAuthenticated}/>
      </Router>
      );
    }
}

ReactDOM.render((
        <App />
      ), document.querySelector('.container'));
