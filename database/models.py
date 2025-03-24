from datetime import datetime

from sqlalchemy import Column, Integer, String, Boolean, DateTime, ForeignKey, Text, BigInteger, JSON
from sqlalchemy.ext.declarative import declarative_base
from sqlalchemy.orm import relationship

Base = declarative_base()


class Admin(Base):
    __tablename__ = 'admins'

    id = Column(Integer, primary_key=True)
    user_id = Column(BigInteger, unique=True, nullable=False)
    username = Column(String(255), nullable=True)
    first_name = Column(String(255), nullable=True)
    last_name = Column(String(255), nullable=True)
    created_at = Column(DateTime, default=datetime.now(datetime.utcnow()))

    def __repr__(self):
        return f"<Admin(user_id={self.user_id}, username={self.username})>"


class BannedUser(Base):
    __tablename__ = 'banned_users'

    id = Column(Integer, primary_key=True)
    user_id = Column(BigInteger, unique=True, nullable=False)
    username = Column(String(255), nullable=True)
    first_name = Column(String(255), nullable=True)
    last_name = Column(String(255), nullable=True)
    banned_at = Column(DateTime, default=datetime.utcnow)
    banned_by = Column(BigInteger, nullable=True)
    reason = Column(String(500), nullable=True)

    def __repr__(self):
        return f"<BannedUser(user_id={self.user_id}, username={self.username})>"


class Meme(Base):
    __tablename__ = 'memes'

    id = Column(Integer, primary_key=True)
    file_id = Column(String(255), unique=True, nullable=False)
    file_type = Column(String(50), nullable=False)
    user_id = Column(BigInteger, nullable=False)
    username = Column(String(255), nullable=True)
    first_name = Column(String(255), nullable=True)
    last_name = Column(String(255), nullable=True)
    created_at = Column(DateTime, default=datetime.utcnow)
    forwarded_to_channel = Column(Boolean, default=False)
    channel_message_id = Column(Integer, nullable=True)

    media_group_id = Column(String(255), nullable=True)

    comments = relationship("MemeComment", back_populates="meme", cascade="all, delete-orphan")

    def __repr__(self):
        return f"<Meme(id={self.id}, file_id={self.file_id}, file_type={self.file_type})>"


class MemeComment(Base):
    __tablename__ = 'meme_comments'

    id = Column(Integer, primary_key=True)
    meme_id = Column(Integer, ForeignKey('memes.id', ondelete='CASCADE'), nullable=False)
    message_id = Column(Integer, nullable=True)
    user_id = Column(BigInteger, nullable=False)
    is_bot = Column(Boolean, default=False)
    content = Column(Text, nullable=False)
    created_at = Column(DateTime, default=datetime.utcnow)

    meme = relationship("Meme", back_populates="comments")

    def __repr__(self):
        return f"<MemeComment(id={self.id}, meme_id={self.meme_id}, is_bot={self.is_bot})>"


class MemeHistory(Base):
    __tablename__ = 'meme_history'

    id = Column(Integer, primary_key=True)
    chat_id = Column(BigInteger, nullable=False)
    meme_id = Column(Integer, ForeignKey('memes.id', ondelete='CASCADE'), nullable=False)
    context = Column(JSON, nullable=True)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)

    def __repr__(self):
        return f"<MemeHistory(id={self.id}, chat_id={self.chat_id}, meme_id={self.meme_id})>"
