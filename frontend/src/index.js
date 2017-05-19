import React, { Component } from 'react';
import ReactDOM from 'react-dom';
import Folder from './components/folders'
import Upload from './components/upload'
import request from 'superagent'
import Login from './components/login'
import Signup from './components/signup'
import User from './components/user'
import { Router, Route, Redirect, Link, browserHistory } from 'react-router'

class App extends Component {
  constructor(props) {
    super(props)
  }

    userIsAuthenticated = (nextState, replace) => {
        var self = this
        if (!document.cookie.startsWith("jwt=")) {
          replace({ pathname: `/login`, state: { nextPathname: nextState.location.pathname }})
        }
    }

    logoutUser = (nextState, replace) => {
      document.cookie = "jwt=;expires=Thu, 01 Jan 1970 00:00:00 GMT";
      browserHistory.push("/login")
    }

    render() {
      return (
      <Router history={browserHistory}>
            <Route path="/" component={Folder} onEnter={this.userIsAuthenticated} />
            <Router path="/signup" component={Signup} />
            <Router path="/login" component={Login} />
            <Router path="/dir" component={Folder} onEnter={this.userIsAuthenticated} />
            <Router path="/logout" onEnter={this.logoutUser} component={Login} />
            <Router path="/user" component={User} />
            <Route path="upload" component={Upload} onEnter={this.userIsAuthenticated}/>
      </Router>
      );
    }
}

ReactDOM.render((
        <App />
      ), document.querySelector('.container'));
