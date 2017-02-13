import React, { Component } from 'react'
import { Router, Route, Link, browserHistory } from 'react-router'
import { WithContext as ReactTags } from 'react-tag-input';

class Search extends Component {
  constructor(props) {
    super(props)
    this.state = { showInputField: false, tags: [] }
  }

  hideInputWhenTagsEmpty = () => {
    if (this.state.tags.length == 0) {
        this.setState({ showInputField: false })
    }
  }

  onClick = () => {
    this.setState({ showInputField: true })
  }

  onBlur = (e) => {
    this.hideInputWhenTagsEmpty()
  }

  handleDelete = (i) => {
      let tags = this.state.tags;
      tags.splice(i, 1);
      this.setState({tags: tags});
      this.hideInputWhenTagsEmpty()
  }

  handleAddition = (tag) => {
      let tags = this.state.tags;
      tags.push({
          id: tags.length + 1,
          text: tag
      });
      this.setState({tags: tags});
  }

  handleDrag = (tag, currPos, newPos) => {
      let tags = this.state.tags;

      // mutate array
      tags.splice(currPos, 1);
      tags.splice(newPos, 0, tag);

      // re-render
      this.setState({ tags: tags });
  }

  handleTagChange = (tags) =>  {
    this.props.tagsCallback(tags)
  }

  render() {
    if (this.state.showInputField) {
      let tags = this.state.tags;
      let suggestions = ["Banana", "Mango", "Pear", "Apricot"]

      this.handleTagChange(tags)

      return (
        <div onBlur={this.onBlur}>
        <ReactTags onBlur={this.onBlur}
                   tags={tags}
                   suggestions={suggestions}
                   handleDelete={this.handleDelete}
                   handleAddition={this.handleAddition}
                   handleDrag={this.handleDrag}
        />
        </div>
      );
    } else {
      return (
        <div onClick={this.onClick}>
          <button type="button" className="btn btn-default">
            <span className="glyphicon glyphicon-search"></span>
          </button>
        </div>
      )
    }
  }
}
export default Search
