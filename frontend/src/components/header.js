import React, { Component } from 'react'
import { Router, Route, Link, browserHistory } from 'react-router'

class Header extends Component {
  render() {
    return (
      <div id="headerMenu">
        <Link to={`/dir`}>NAVIGATE</Link>
        <Link to={`/upload`}>UPLOAD</Link>
      </div>
    );
  }
}
export default Header
