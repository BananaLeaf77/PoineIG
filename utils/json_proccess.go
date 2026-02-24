package utils

import (
	"encoding/json"
	"fmt"
	"os"
)

func ReadAllFollowers(basePath string) []byte {
	var allFollowers []map[string]interface{}
	for i := 1; i <= 10; i++ {
		path := fmt.Sprintf("%s/followers_%d.json", basePath, i)
		data, err := os.ReadFile(path)
		if err != nil {
			break
		}
		var chunk []map[string]interface{}
		json.Unmarshal(data, &chunk)
		allFollowers = append(allFollowers, chunk...)
		fmt.Printf("Loaded followers_%d.json (%d entries)\n", i, len(chunk))
	}
	result, _ := json.Marshal(allFollowers)
	return result
}

func CompareUsernames(followingData []byte, followersData []byte) []string {
	var followingWrapper struct {
		RelationshipsFollowing []struct {
			Title string `json:"title"`
		} `json:"relationships_following"`
	}
	json.Unmarshal(followingData, &followingWrapper)

	var followersList []struct {
		StringListData []struct {
			Value string `json:"value"`
		} `json:"string_list_data"`
	}
	json.Unmarshal(followersData, &followersList)

	followersSet := make(map[string]bool)
	for _, f := range followersList {
		if len(f.StringListData) > 0 {
			followersSet[f.StringListData[0].Value] = true
		}
	}

	var toUnfollow []string
	for _, f := range followingWrapper.RelationshipsFollowing {
		if !followersSet[f.Title] {
			toUnfollow = append(toUnfollow, f.Title)
		}
	}

	fmt.Printf("You follow: %d | Followers: %d | Not following back: %d\n",
		len(followingWrapper.RelationshipsFollowing),
		len(followersList),
		len(toUnfollow),
	)
	return toUnfollow
}
