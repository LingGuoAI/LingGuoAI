package jobs

import (
	"encoding/json"
	"testing"
)

func TestAIStoryboardAcceptsSnakeCaseFields(t *testing.T) {
	input := `{"shot_number":2,"shot_type":"近景","camera_movement":"推镜","scene_id":7,"visual_desc":"完整画面","audio_prompt":"风声","duration_sec":6,"character_ids":[11],"prop_ids":[12],"result":"门被推开"}`
	var shot AIStoryboard
	if err := json.Unmarshal([]byte(input), &shot); err != nil {
		t.Fatal(err)
	}
	if shot.SequenceNo != 2 || shot.ShotType != "近景" || shot.CameraMovement != "推镜" || shot.SceneID == nil || *shot.SceneID != 7 {
		t.Fatalf("basic fields were not normalized: %+v", shot)
	}
	if shot.VisualDesc != "完整画面" || shot.AudioPrompt != "风声" || shot.DurationSec != 6 || shot.Result != "门被推开" {
		t.Fatalf("content fields were not normalized: %+v", shot)
	}
	if len(shot.CharacterIDs) != 1 || shot.CharacterIDs[0] != 11 || len(shot.PropIDs) != 1 || shot.PropIDs[0] != 12 {
		t.Fatalf("relation fields were not normalized: %+v", shot)
	}
}

func TestAIStoryboardCamelCaseTakesPrecedence(t *testing.T) {
	input := `{"sequenceNo":3,"shot_number":9,"durationSec":5,"duration_sec":8}`
	var shot AIStoryboard
	if err := json.Unmarshal([]byte(input), &shot); err != nil {
		t.Fatal(err)
	}
	if shot.SequenceNo != 3 || shot.DurationSec != 5 {
		t.Fatalf("camelCase fields must take precedence: %+v", shot)
	}
}
