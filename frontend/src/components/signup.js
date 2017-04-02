import React, { Component } from 'react'
import request from 'superagent'

class Signup extends Component {
  constructor(props) {
    super(props)

    this.state = {password1: "", password2: ""}
  }

  onSubmit = (e) => {
    e.preventDefault()
    var self = this;
    request
      .post('/account/signup')
      .type('form')
      .send({password: document.forms.item(0)[1].value })
      .end(function(error, response){
        if(error) {
          if (response.statusCode == 403) {
            self.setState({"wrongPassword": true})
          }
        } else {
          self.setState({"wrongPassword": false})

          var token = JSON.parse(response.text)["token"]
          document.cookie = "jwt=" + token;

          if (self.props.location.state != null) {
            browserHistory.push(self.props.location.state.nextPathname)
          } else {
            browserHistory.push(`/dir`)
          }
        }
      });
  }

  verifyPassword1 = (e) => {
    this.setState({password1: e.target.value}, function(){
      console.log(this.state)
      console.log(this.state.password1 == this.state.password2)
    })
  }

  verifyPassword2 = (e) => {
    this.setState({password2: e.target.value}, function(){
      console.log(this.state)
      console.log(this.state.password1 == this.state.password2)
    })
  }

  render () {
    return (
       <form className="form-horizontal"  onSubmit={this.onSubmit}>
         <fieldset>
           <legend>Signup</legend>
           <div className="form-group">
             <label className="col-md-4 control-label" htmlFor="login">Username</label>
             <div className="col-md-4">
               <input id="login" name="login" className="form-control input-md" value="admin" disabled readOnly/>
             </div>
           </div>
           <div className="form-group">
             <label className="col-md-4 control-label" htmlFor="password">Password</label>
             <div className="col-md-4">
               <input id="password1" name="password" type="password" className="form-control input-md" onChange={this.verifyPassword1} required />
             </div>
           </div>
           <div className="form-group">
             <label className="col-md-4 control-label" htmlFor="password">Password (repeat)</label>
             <div className="col-md-4">
               <input id="password2" name="password" type="password" className="form-control input-md" onChange={this.verifyPassword2} required />
             </div>
           </div>
           <div className="form-group">
             <label className="col-md-4 control-label" htmlFor="singlebutton" />
             <div className="col-md-4">
             <input type="submit" id="singlebutton" name="singlebutton" className="center btn btn-primary" value="Signup" disabled/>
           </div>
           </div>
         </fieldset>
       </form>
     );
   }
}

export default Signup
