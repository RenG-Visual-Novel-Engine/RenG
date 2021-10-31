package evaluator

import (
	"RenG/src/config"
	"RenG/src/core"
	"RenG/src/lang/ast"
	"RenG/src/lang/object"
)

func evalScreenExpression(se *ast.ScreenExpression, env *object.Environment) object.Object {
	env.Set(se.Name.Value, &object.Screen{Name: se.Name, Body: se.Body})

	return nil
}

func evalLabelExpression(le *ast.LabelExpression, env *object.Environment) object.Object {
	env.Set(le.Name.Value, &object.Label{Name: le.Name, Body: le.Body})

	return nil
}

func evalImageExpression(ie *ast.ImageExpression, env *object.Environment) object.Object {
	rootObj := Eval(ie.Path, env)

	if path, ok := rootObj.(*object.String); ok {
		texture, suc := config.Renderer.LoadFromFile(config.Path + path.Value)
		if !suc {
			return newError("Failed Load %s Texture", path.Value)
		}

		config.TextureList.Set(ie.Name.Value, texture)
	} else {
		return newError("Path is not string")
	}

	return nil
}

func evalVideoExpression(ve *ast.VideoExpression, env *object.Environment) object.Object {
	var video object.VideoObject
	path := Eval(ve.Info["path"], env)
	size := Eval(ve.Info["size"], env)

	video.Video = core.OpenVideo(config.Path+path.(*object.String).Value,
		int(size.(*object.Array).Elements[0].(*object.Integer).Value),
		int(size.(*object.Array).Elements[1].(*object.Integer).Value),
	)

	video.Texture = config.Renderer.CreateTexture(video.Video.W, video.Video.H)

	config.VideoList.Set(ve.Name.Value, &video)

	return nil
}

func evalTransformExpression(te *ast.TransformExpression, env *object.Environment) object.Object {
	env.Set(te.Name.Value, &object.Transform{Name: te.Name, Body: te.Body})

	return nil
}