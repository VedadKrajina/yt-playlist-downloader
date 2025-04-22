import yt_dlp

def download_playlist(playlist_url):
    """
    Downloads all videos from the specified YouTube playlist as MP3 files.
    
    Args:
        playlist_url (str): The URL of the YouTube playlist to download.
    """
    ydl_opts = {
        'outtmpl': '/home/vk15/Music/%(playlist)s/%(playlist_index)s - %(title)s.%(ext)s',  # Save audio in /home/vk15/Music/
        'format': 'bestaudio/best',  # Download the best available audio
        'noplaylist': False,  # Ensure the entire playlist is downloaded
        'postprocessors': [{  # Convert audio to MP3
            'key': 'FFmpegExtractAudio',
            'preferredcodec': 'mp3',
            'preferredquality': '192',  # Set audio quality to 192kbps
        }],
        'geo_bypass': True,  # Bypass geographic restrictions
        'ignoreerrors': True,  # Skip errors and continue
    }
    try:
        with yt_dlp.YoutubeDL(ydl_opts) as ydl:
            ydl.download([playlist_url])
    except Exception as e:
        handle_errors(e)

def download_video(video_url):
    """
    Downloads a single video from YouTube as an MP3 file.
    
    Args:
        video_url (str): The URL of the YouTube video to download.
    """
    ydl_opts = {
        'outtmpl': '%(title)s.%(ext)s',  # Save audio with its title as the filename
        'format': 'bestaudio/best',  # Download the best available audio
        'geo_bypass': True,  # Bypass geographic restrictions
        'ignoreerrors': True,  # Skip errors and continue
        'postprocessors': [{  # Convert audio to MP3
            'key': 'FFmpegExtractAudio',
            'preferredcodec': 'mp3',
            'preferredquality': '192',  # Set audio quality to 192kbps
        }],
    }
    try:
        with yt_dlp.YoutubeDL(ydl_opts) as ydl:
            ydl.download([video_url])
    except Exception as e:
        handle_errors(e)

def handle_errors(error):
    """
    Handles errors that occur during the download process.
    
    Args:
        error (Exception): The error that occurred.
    """
    print(f"An error occurred: {error}")

if __name__ == "__main__":
    # Example usage
    playlist_url = input("Enter the YouTube playlist URL: ")
    download_playlist(playlist_url)