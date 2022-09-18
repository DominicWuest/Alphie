gen:
	# Image Generation
	protoc --go-grpc_out=image_generation_server/image_generation_pb --go-grpc_opt=paths=source_relative \
			--go-grpc_out=../bot/commands/image_generation --go-grpc_opt=paths=source_relative \
			\
			--go_out=image_generation_server/image_generation_pb --go_opt=paths=source_relative \
			--go_out=../bot/commands/image_generation --go_opt=paths=source_relative \
			\
			-I image_generation_server/image_generation_pb image_generation.proto	

	# Lecture Clips
	protoc --go-grpc_out=lecture_clip_server/lecture_clip_pb --go-grpc_opt=paths=source_relative \
			--go-grpc_out=../bot/commands/clip --go-grpc_opt=paths=source_relative \
			\
			--go_out=lecture_clip_server/lecture_clip_pb --go_opt=paths=source_relative \
			--go_out=../bot/commands/clip --go_opt=paths=source_relative \
			\
			-I lecture_clip_server/lecture_clip_pb lecture_clip.proto	