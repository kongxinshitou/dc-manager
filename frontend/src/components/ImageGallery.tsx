import { useState } from 'react'
import { Upload, Button, Image, message, Popconfirm } from 'antd'
import { PlusOutlined, DeleteOutlined } from '@ant-design/icons'
import { uploadInspectionImages, deleteInspectionImage, getImageUrl, type InspectionImage } from '../api'

interface ImageGalleryProps {
  inspectionId: number
  images: InspectionImage[]
  canUpload?: boolean
  canDelete?: boolean
  onChange: (images: InspectionImage[]) => void
}

export default function ImageGallery({ inspectionId, images, canUpload, canDelete, onChange }: ImageGalleryProps) {
  const [uploading, setUploading] = useState(false)

  const handleUpload = async (fileList: File[]) => {
    if (images.length + fileList.length > 9) {
      message.warning('每条巡检记录最多上传9张图片')
      return
    }
    setUploading(true)
    try {
      const newImages = await uploadInspectionImages(inspectionId, fileList)
      onChange([...images, ...newImages])
      message.success(`上传成功 ${newImages.length} 张图片`)
    } catch (err: any) {
      message.error(err.response?.data?.error || '上传失败')
    } finally {
      setUploading(false)
    }
  }

  const handleDelete = async (imageId: number) => {
    try {
      await deleteInspectionImage(inspectionId, imageId)
      onChange(images.filter(img => img.id !== imageId))
      message.success('图片已删除')
    } catch {
      message.error('删除失败')
    }
  }

  return (
    <div>
      <div style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fill, minmax(140px, 1fr))',
        gap: 12,
        marginBottom: canUpload ? 12 : 0,
      }}>
        {images.map(img => (
          <div key={img.id} style={{ position: 'relative' }}>
            <Image
              src={getImageUrl(img.file_path)}
              alt={img.file_name}
              style={{ width: '100%', height: 120, objectFit: 'cover', borderRadius: 6 }}
              preview={{
                mask: <div style={{ fontSize: 12 }}>{img.file_name}</div>,
              }}
            />
            {canDelete && (
              <Popconfirm title="确认删除此图片？" onConfirm={() => handleDelete(img.id)}>
                <Button
                  type="primary"
                  danger
                  size="small"
                  shape="circle"
                  icon={<DeleteOutlined />}
                  style={{ position: 'absolute', top: 4, right: 4, minWidth: 24, width: 24, height: 24 }}
                />
              </Popconfirm>
            )}
            <div style={{
              fontSize: 11, color: '#999', marginTop: 4,
              overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
            }}>
              {img.file_name}
            </div>
          </div>
        ))}
      </div>

      {canUpload && images.length < 9 && (
        <Upload
          accept="image/*"
          showUploadList={false}
          multiple
          beforeUpload={(_file, fileList) => {
            handleUpload(fileList as unknown as File[])
            return false
          }}
        >
          <Button icon={<PlusOutlined />} loading={uploading}>
            上传图片 ({images.length}/9)
          </Button>
        </Upload>
      )}
    </div>
  )
}
